package storage

import (
	"context"
	"database/sql"
	"encoding/json"
	"net"
	"strings"
	"sync"
	"time"

	"github.com/pkg/errors"

	"github.com/asecurityteam/asset-inventory-api/pkg/domain"
)

const (
	added   = "ADDED"   // one of the network event types we track
	deleted = "DELETED" // deleted network event
)

// Query to find resource by private IP using v2 schema
const resourceByPrivateIPQuery = `select * from get_resource_by_private_ip($1, $2)`

// Query to find resource by public IP using v2 schema
const resourceByPublicIPQuery = `select * from get_resource_by_public_ip($1, $2)`

// Query to find resource by hostname using v2 schema
const resourceByHostnameQuery = `select * from get_resource_by_hostname($1, $2)`

// Query to find resource by ARN ID
const resourceByARNIDQuery = `select * from get_resource_by_arn_id($1, $2)`

// Query to find owner and champions by account ID, which is auto-increment primary key
const ownerByAccountIDQuery = `select * from get_owner_and_champions_by_account_id($1)`

const insertPersonQuery = `
INSERT INTO person(login, email, name, valid)
VALUES ($1, $2, $3, $4)
ON CONFLICT(login) DO UPDATE
SET email=$2, name=$3, valid=$4;
`

const resourceIDQuery = `
SELECT ar.id FROM aws_resource ar
	LEFT JOIN aws_region reg ON reg.id = ar.aws_region_id 
	LEFT JOIN aws_account aa ON aa.id = ar.aws_account_id
	WHERE ar.arn_id = $1
		AND aa.account = $2
		AND reg.region = $3;
`

// DB represents a convenient database abstraction layer
type DB struct {
	sqldb               *sql.DB // this is a unit test seam
	once                sync.Once
	now                 func() time.Time // unit test seam
	defaultPartitionTTL int
}

var privateIPNetworks = []net.IPNet{
	{
		IP:   net.IPv4(192, 168, 1, 0),
		Mask: net.IPv4Mask(255, 255, 0, 0),
	},
	{
		IP:   net.IPv4(172, 16, 0, 0),
		Mask: net.IPv4Mask(255, 240, 0, 0),
	},
	{
		IP:   net.IPv4(10, 0, 0, 0),
		Mask: net.IPv4Mask(255, 0, 0, 0),
	},
}

// Init initializes a connection to a Postgres database according to the environment variables POSTGRES_HOST, POSTGRES_PORT, POSTGRES_USER, POSTGRES_PASSWORD, POSTGRES_DATABASE
func (db *DB) Init(ctx context.Context, url string, partitionTTL int) error {
	var initErr error
	db.once.Do(func() {

		db.defaultPartitionTTL = partitionTTL

		if db.now == nil {
			db.now = time.Now
		}

		if db.sqldb == nil {
			pgdb, err := sql.Open("postgres", url)
			if err != nil {
				initErr = err
				return // from the unnamed once.Do function
			}

			db.sqldb = pgdb
			err = db.ping()
			if err != nil {
				initErr = err
				return // from the unnamed once.Do function
			}

		}
	})
	return initErr
}

// nb this is intentionally private for use by DB code to avoid confusion with fully-functioning SchemaVersionManager
func (db *DB) getSchemaVersion(ctx context.Context) (uint, error) {
	const versionQuery = `select version from schema_migrations`
	ver := uint(0)
	err := db.sqldb.QueryRowContext(ctx, versionQuery).Scan(&ver)
	if err != nil && err != sql.ErrNoRows { //no rows as version 0
		return 0, err
	}
	return ver, nil
}

// use function's intent is to close the existing connection (pgdb parameter) and open
// a new one against the desired psqlInfo connection string
func (db *DB) use(psqlInfo string) error {

	err := db.sqldb.Close()
	if err != nil {
		return err
	}
	pgdb, err := sql.Open("postgres", psqlInfo)
	if err != nil {
		return err
	}

	db.sqldb = pgdb

	return nil
}

// ping is required for Postgres connection to be fully established
func (db *DB) ping() error {

	err := db.sqldb.Ping()
	if err != nil {
		return err
	}

	return nil
}

// Store an implementation of the Storage interface that records to a database
func (db *DB) Store(ctx context.Context, cloudAssetChanges domain.CloudAssetChanges) error {
	tx, err := db.sqldb.Begin()
	if err != nil {
		return err
	}
	if err = db.ensureResourceExists(ctx, cloudAssetChanges, tx); err == nil {
		err = db.applyChanges(ctx, cloudAssetChanges, tx)
	}
	if err != nil {
		if rollbackErr := tx.Rollback(); rollbackErr != nil {
			return errors.Wrap(rollbackErr, err.Error()) // so we don't lose the original error
		}
		return err
	}
	if err = tx.Commit(); err != nil {
		return err
	}
	return nil
}

func (db *DB) applyChanges(ctx context.Context, cloudAssetChanges domain.CloudAssetChanges, tx *sql.Tx) error {
	var err error
	arnID := resIDFromARN(cloudAssetChanges.ARN)
	resourceID, err := db.getResourceID(ctx, tx, arnID, cloudAssetChanges.Region, cloudAssetChanges.AccountID)
	if err != nil {
		return err
	}
	for _, val := range cloudAssetChanges.Changes {
		for _, ip := range val.PrivateIPAddresses {
			if strings.EqualFold(added, val.ChangeType) {
				err = db.assignPrivateIP(ctx, tx, resourceID, ip, cloudAssetChanges.ChangeTime)
			} else {
				err = db.releasePrivateIP(ctx, tx, resourceID, ip, cloudAssetChanges.ChangeTime)
			}
			if err != nil {
				return err
			}
		}
		for _, ip := range val.PublicIPAddresses {
			for _, hostname := range val.Hostnames { //TODO look very closely into awsconfig-tranformerd logic for this
				if strings.EqualFold(added, val.ChangeType) {
					err = db.assignPublicIP(ctx, tx, resourceID, ip, hostname, cloudAssetChanges.ChangeTime)
				} else {
					err = db.releasePublicIP(ctx, tx, resourceID, ip, hostname, cloudAssetChanges.ChangeTime)
				}
				if err != nil {
					return err
				}
			}
		}
		for _, res := range val.RelatedResources {
			if strings.EqualFold(added, val.ChangeType) {
				err = db.assignResourceRelationship(ctx, tx, arnID, res, cloudAssetChanges.ChangeTime)
			} else {
				err = db.releaseResourceRelationship(ctx, tx, arnID, res, cloudAssetChanges.ChangeTime)
			}
			if err != nil {
				return err
			}
		}
	}
	return nil
}

func (db *DB) ensureResourceExists(ctx context.Context, cloudAssetChanges domain.CloudAssetChanges, tx *sql.Tx) error {
	// reading pre-requisite to understand this query - https://www.postgresql.org/docs/current/queries-with.html
	const createResourceQuery string = `
with sel as (
    select val.arn_id,
           val.region,
           aws_region.id        as aws_region_id,
           val.account,
           aws_account.id       as aws_account_id,
           val.resource_type,
           aws_resource_type.id as aws_resource_type_id,
           val.meta
    from (
             values ($1::text, $2::text, $3::text, $4::text, $5::jsonb)
         ) val (arn_id, region, account, resource_type, meta)
             left join aws_region using (region)
             left join aws_account using (account)
             left join aws_resource_type using (resource_type)),
     ins_aws_region as (
         insert into aws_region (region)
             select distinct region from sel where aws_region_id is null
             returning id as aws_region_id, region
     ),
     ins_aws_account as (
         insert into aws_account (account)
             select distinct account from sel where aws_account_id is null
             returning id as aws_account_id, account
     ),
     ins_aws_resource_type as (
         insert into aws_resource_type (resource_type)
             select distinct resource_type from sel where aws_resource_type_id is null
             returning id as aws_resource_type_id, resource_type
     )
insert
into aws_resource (arn_id, aws_region_id, aws_account_id, aws_resource_type_id, meta)
select sel.arn_id,
       coalesce(sel.aws_region_id, ins_aws_region.aws_region_id),
       coalesce(sel.aws_account_id, ins_aws_account.aws_account_id),
       coalesce(sel.aws_resource_type_id, ins_aws_resource_type.aws_resource_type_id),
       sel.meta
from sel
         left join ins_aws_region using (region)
         left join ins_aws_account using (account)
         left join ins_aws_resource_type using (resource_type)
on conflict do nothing
`
	tagsBytes, _ := json.Marshal(cloudAssetChanges.Tags) // an error here is not possible considering json.Marshal is taking a simple map or nil
	if _, err := tx.ExecContext(ctx,
		createResourceQuery,
		resIDFromARN(cloudAssetChanges.ARN),
		cloudAssetChanges.Region,
		cloudAssetChanges.AccountID,
		cloudAssetChanges.ResourceType,
		tagsBytes); err != nil {
		return err
	}
	return nil
}

// extract unique resource-type/resource-id from full ARN format
// it is always the part after account-id:
// https://docs.aws.amazon.com/general/latest/gr/aws-arns-and-namespaces.html
func resIDFromARN(ARN string) string {
	parts := strings.SplitN(ARN, ":", 6)
	resourceID := parts[len(parts)-1]
	if strings.HasPrefix(resourceID, "loadbalancer/app") {
		return resourceID[13:]
	}
	parts = strings.SplitAfterN(resourceID, "/", -1)
	return parts[len(parts)-1]
}

// FetchAll gets all the assets present at the specified time
func (db *DB) FetchAll(ctx context.Context, when time.Time, count uint, offset uint, typeFilter string) ([]domain.CloudAssetDetails, error) {
	return nil, errors.New("bulk export API is not available")
}

// FetchByHostname gets the assets who have hostname at the specified time
func (db *DB) FetchByHostname(ctx context.Context, when time.Time, hostname string) ([]domain.CloudAssetDetails, error) {
	return db.runLookupQuery(ctx, false, resourceByHostnameQuery, hostname, when)
}

// FetchByIP gets the assets who have IP address at the specified time
func (db *DB) FetchByIP(ctx context.Context, when time.Time, ipAddress string) ([]domain.CloudAssetDetails, error) {
	ipaddr := net.ParseIP(ipAddress)
	if ipaddr == nil {
		return nil, errors.New("invalid IP address")
	}
	if isPrivateIP(ipaddr) {
		return db.runLookupQuery(ctx, true, resourceByPrivateIPQuery, ipAddress, when)
	}
	return db.runLookupQuery(ctx, false, resourceByPublicIPQuery, ipAddress, when)
}

func isPrivateIP(ip net.IP) bool {
	for _, net := range privateIPNetworks {
		if net.Contains(ip) {
			return true
		}
	}
	return false
}

func (db *DB) runLookupQuery(ctx context.Context, isPrivateIP bool, query string, args ...interface{}) ([]domain.CloudAssetDetails, error) {
	rows, err := db.sqldb.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	cloudAssetDetails := make([]domain.CloudAssetDetails, 0)
	tempMap := make(map[string]*domain.CloudAssetDetails)
	for rows.Next() {
		var row domain.CloudAssetDetails
		var account domain.AccountOwner

		var metaBytes []byte
		var hostname sql.NullString
		var ipAddress string // no need for sql.NullBool as the DB column is guaranteed a value
		var accountID int
		var chLogin *string
		var chEmail *string
		var chName *string
		var chValid *bool

		if isPrivateIP {
			err = rows.Scan(&ipAddress, &row.ARN, &metaBytes, &row.Region, &row.ResourceType, &row.AccountID,
				&accountID, &account.AccountID, &account.Owner.Login, &account.Owner.Email,
				&account.Owner.Name, &account.Owner.Valid, &chLogin, &chEmail, &chName, &chValid)
		} else {
			err = rows.Scan(&ipAddress, &hostname, &row.ARN, &metaBytes, &row.Region, &row.ResourceType, &row.AccountID,
				&accountID, &account.AccountID, &account.Owner.Login, &account.Owner.Email,
				&account.Owner.Name, &account.Owner.Valid, &chLogin, &chEmail, &chName, &chValid)
		}
		if err != nil {
			return nil, err
		}

		if metaBytes != nil {
			var i map[string]string
			_ = json.Unmarshal(metaBytes, &i) // we already checked for nil, and the DB column is JSONB; no need for err check here
			row.Tags = i
		}
		if tempMap[row.ARN] == nil {
			tempMap[row.ARN] = &row
		}

		if tempMap[row.ARN].AccountOwner.Champions == nil {
			tempMap[row.ARN].AccountOwner = domain.AccountOwner{
				AccountID: account.AccountID,
				Owner: domain.Person{
					Name:  account.Owner.Name,
					Login: account.Owner.Login,
					Email: account.Owner.Email,
					Valid: account.Owner.Valid,
				},
				Champions: make([]domain.Person, 0),
			}
		}

		champion := domain.Person{
			Login: chLogin,
			Email: chEmail,
			Name:  chName,
			Valid: chValid,
		}

		found := false
		for _, val := range tempMap[row.ARN].AccountOwner.Champions {
			if val.Email != nil && chEmail != nil && strings.EqualFold(*val.Email, *chEmail) {
				found = true
				break
			}
		}
		if !found {
			tempMap[row.ARN].AccountOwner.Champions = append(tempMap[row.ARN].AccountOwner.Champions, champion)
		}

		found = false
		if hostname.Valid {
			for _, val := range tempMap[row.ARN].Hostnames {
				if strings.EqualFold(val, hostname.String) {
					found = true
					break
				}
			}
			if !found {
				tempMap[row.ARN].Hostnames = append(tempMap[row.ARN].Hostnames, hostname.String)
			}
		}
		found = false
		var ipAddresses *[]string
		if !isPrivateIP {
			ipAddresses = &tempMap[row.ARN].PublicIPAddresses
		} else {
			ipAddresses = &tempMap[row.ARN].PrivateIPAddresses
		}
		for _, val := range *ipAddresses {
			if strings.EqualFold(val, ipAddress) {
				found = true
				break
			}
		}
		if !found {
			newArray := append(*ipAddresses, ipAddress)
			if !isPrivateIP {
				tempMap[row.ARN].PublicIPAddresses = newArray
			} else {
				tempMap[row.ARN].PrivateIPAddresses = newArray
			}
		}
	}

	rows.Close() // no need to capture the returned error since we check rows.Err() immediately:

	if err = rows.Err(); err != nil {
		return nil, err
	}

	for _, val := range tempMap {
		cloudAssetDetails = append(cloudAssetDetails, *val)
	}

	return cloudAssetDetails, nil
}

// FetchByResourceID gets the assets who have resource ID at the specified time
func (db *DB) FetchByResourceID(ctx context.Context, when time.Time, resID string) ([]domain.CloudAssetDetails, error) {
	rows, err := db.sqldb.QueryContext(ctx, resourceByARNIDQuery, resID, when)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	cloudAssetDetails := make([]domain.CloudAssetDetails, 0)

	tempPublicIPMap := make(map[string]struct{})
	tempPrivateIPMap := make(map[string]struct{})
	tempHostnameMap := make(map[string]struct{})
	tempChampionsMap := make(map[string]domain.Person)

	var account domain.AccountOwner
	var asset domain.CloudAssetDetails
	var accountID int
	hasTag := false
	empty := true
	for rows.Next() {
		empty = false // there is no way to check for size of SQL result set in golang
		var privateIPAddress sql.NullString
		var publicIPAddress sql.NullString
		var hostname sql.NullString
		var metaBytes []byte
		var chLogin *string
		var chEmail *string
		var chName *string
		var chValid *bool
		if err = rows.Scan(&privateIPAddress, &publicIPAddress, &hostname, &asset.ResourceType, &asset.AccountID,
			&asset.Region, &metaBytes, &accountID, &account.AccountID, &account.Owner.Login, &account.Owner.Email,
			&account.Owner.Name, &account.Owner.Valid, &chLogin, &chEmail, &chName, &chValid); err != nil {
			return nil, err
		}

		if privateIPAddress.Valid {
			tempPrivateIPMap[privateIPAddress.String] = struct{}{}
		}
		if publicIPAddress.Valid {
			tempPublicIPMap[publicIPAddress.String] = struct{}{}
			if hostname.Valid {
				tempHostnameMap[hostname.String] = struct{}{}
			}
		}

		if metaBytes != nil && !hasTag {
			var i map[string]string
			_ = json.Unmarshal(metaBytes, &i) // we already checked for nil, and the DB column is JSONB; no need for err check here
			asset.Tags = i
			hasTag = true
		}

		if chLogin != nil {
			tempChampionsMap[*chLogin] = domain.Person{
				Login: chLogin,
				Email: chEmail,
				Name:  chName,
				Valid: chValid,
			}
		}
	}
	rows.Close()
	if err = rows.Err(); err != nil {
		return nil, err
	}

	if empty { // we got 0 rows, nothing to process
		return cloudAssetDetails, nil
	}

	for ip := range tempPrivateIPMap {
		asset.PrivateIPAddresses = append(asset.PrivateIPAddresses, ip)
	}
	for ip := range tempPublicIPMap {
		asset.PublicIPAddresses = append(asset.PublicIPAddresses, ip)
	}
	for hostname := range tempHostnameMap {
		asset.Hostnames = append(asset.Hostnames, hostname)
	}
	asset.ARN = resID

	asset.AccountOwner = domain.AccountOwner{
		AccountID: account.AccountID,
		Owner: domain.Person{
			Name:  account.Owner.Name,
			Login: account.Owner.Login,
			Email: account.Owner.Email,
			Valid: account.Owner.Valid,
		},
		Champions: make([]domain.Person, 0),
	}
	for champion := range tempChampionsMap {
		asset.AccountOwner.Champions = append(asset.AccountOwner.Champions, tempChampionsMap[champion])
	}
	cloudAssetDetails = append(cloudAssetDetails, asset)
	return cloudAssetDetails, err
}

func (db *DB) assignPrivateIP(ctx context.Context, tx *sql.Tx, resourceID int, ip string, when time.Time) error {
	const assignPrivateIPQueryUpdate = `
update aws_private_ip_assignment
set not_before = $1
where private_ip = $2
  and not_before = to_timestamp(0)
  and not_after > $1
  and aws_resource_id = $3;`

	const assignPrivateIPQueryInsert = `
insert into aws_private_ip_assignment
    (not_before, private_ip, aws_resource_id)
values ($1, $2, $3) on conflict do nothing ;`

	res, err := tx.ExecContext(ctx, assignPrivateIPQueryUpdate, when, ip, resourceID)
	if err != nil {
		return err
	}
	changedRows, err := res.RowsAffected()
	if err != nil {
		return err
	}
	if changedRows != 0 {
		return nil
	}
	_, err = tx.ExecContext(ctx, assignPrivateIPQueryInsert, when, ip, resourceID)
	return err
}

func (db *DB) releasePrivateIP(ctx context.Context, tx *sql.Tx, resourceID int, ip string, when time.Time) error {
	//we use to_timestamp(0) for release events w/o known assignment as the start of epoch - 1970-01-01 is known
	//to not have AWS resources by definition
	//this way we can find unbalanced events while avoiding nullable not_before
	//which provides minimal support for out-of-order events
	const releasePrivateIPQueryUpdate = `
update aws_private_ip_assignment
set not_after=$1
where private_ip = $2
  and aws_resource_id = $3
  and not_after is null ;`

	const releasePrivateIPQueryInsert = `
insert into aws_private_ip_assignment
    (not_before, not_after, private_ip, aws_resource_id)
values (to_timestamp(0), $1, $2, $3) on conflict do nothing ;`

	res, err := tx.ExecContext(ctx, releasePrivateIPQueryUpdate, when, ip, resourceID)
	if err != nil {
		return err
	}
	changedRows, err := res.RowsAffected()
	if err != nil {
		return err
	}
	if changedRows != 0 {
		return nil
	}
	_, err = tx.ExecContext(ctx, releasePrivateIPQueryInsert, when, ip, resourceID)
	return err
}

func (db *DB) getResourceID(ctx context.Context, tx *sql.Tx, arnID string, region string, accountID string) (int, error) {
	row := tx.QueryRowContext(ctx, resourceIDQuery, arnID, accountID, region)
	var resourceID int
	if err := row.Scan(&resourceID); err != nil {
		return -1, err
	}
	return resourceID, nil
}

func (db *DB) assignPublicIP(ctx context.Context, tx *sql.Tx, resourceID int, ip string, hostname string, when time.Time) error {
	const assignPublicIPQueryUpdate = `
update aws_public_ip_assignment
set not_before = $1
where public_ip = $2
  and not_before = to_timestamp(0)
  and not_after > $1
  and aws_resource_id = $3
  and aws_hostname = $4`

	const assignPublicIPQueryInsert = `
insert into aws_public_ip_assignment
    (not_before, public_ip, aws_resource_id, aws_hostname)
values ($1, $2, $3, $4) on conflict do nothing`

	res, err := tx.ExecContext(ctx, assignPublicIPQueryUpdate, when, ip, resourceID, hostname)
	if err != nil {
		return err
	}
	changedRows, err := res.RowsAffected()
	if err != nil {
		return err
	}
	if changedRows != 0 {
		return nil
	}
	_, err = tx.ExecContext(ctx, assignPublicIPQueryInsert, when, ip, resourceID, hostname)
	return err
}

func (db *DB) releasePublicIP(ctx context.Context, tx *sql.Tx, resourceID int, ip string, hostname string, when time.Time) error {
	//we use to_timestamp(0) for release events w/o known assignment as the start of epoch - 1970-01-01 is known
	//to NOT have AWS resources by definition
	//this way we can find unbalanced events while avoiding nullable not_before
	//which provides minimal support for out-of-order events
	const releasePublicIPQueryUpdate = `
        update aws_public_ip_assignment
        set not_after=$1
        where public_ip = $2
          and aws_resource_id = $3
          and aws_hostname = $4
          and not_after is null`

	const releasePublicIPQueryInsert = `
            insert into aws_public_ip_assignment
                (not_before, not_after, public_ip, aws_resource_id, aws_hostname)
            values (to_timestamp(0), $1, $2, $3, $4) on conflict do nothing `

	res, err := tx.ExecContext(ctx, releasePublicIPQueryUpdate, when, ip, resourceID, hostname)
	if err != nil {
		return err
	}
	changedRows, err := res.RowsAffected()
	if err != nil {
		return err
	}
	if changedRows != 0 {
		return nil
	}
	_, err = tx.ExecContext(ctx, releasePublicIPQueryInsert, when, ip, resourceID, hostname)
	return err
}

func (db *DB) assignResourceRelationship(ctx context.Context, tx *sql.Tx, arnID string, resource string, when time.Time) error {
	const assignResourceRelationshipQueryUpdate = `
update aws_resource_relationship
set not_before = $1
where related_arn_id = $2
  and not_before = to_timestamp(0)
  and not_after > $1
  and arn_id = $3;`

	const assignResourceRelationshipQueryInsert = `
insert into aws_resource_relationship
    (not_before, related_arn_id, arn_id)
values ($1, $2, $3) on conflict do nothing ;`

	res, err := tx.ExecContext(ctx, assignResourceRelationshipQueryUpdate, when, resource, arnID)
	if err != nil {
		return err
	}
	changedRows, err := res.RowsAffected()
	if err != nil {
		return err
	}
	if changedRows != 0 {
		return nil
	}
	_, err = tx.ExecContext(ctx, assignResourceRelationshipQueryInsert, when, resource, arnID)
	return err
}

func (db *DB) releaseResourceRelationship(ctx context.Context, tx *sql.Tx, arnID string, resource string, when time.Time) error {
	const releaseResourceRelationshipQueryUpdate = `
update aws_resource_relationship
set not_after=$1
where related_arn_id = $2
  and arn_id = $3
  and not_after is null;`

	const releaseResourceRelationshipQueryInsert = `
insert into aws_resource_relationship
    (not_before, not_after, related_arn_id, arn_id)
values (to_timestamp(0), $1, $2, $3) on conflict do nothing ;`

	res, err := tx.ExecContext(ctx, releaseResourceRelationshipQueryUpdate, when, resource, arnID)
	if err != nil {
		return err
	}
	changedRows, err := res.RowsAffected()
	if err != nil {
		return err
	}
	if changedRows != 0 {
		return nil
	}
	_, err = tx.ExecContext(ctx, releaseResourceRelationshipQueryInsert, when, resource, arnID)
	return err
}

// StoreAccountOwner is an implementation of AccountOwnerStorer interface that saves account ID, its owner and champions of the account to a database
func (db *DB) StoreAccountOwner(ctx context.Context, accountOwner domain.AccountOwner) error {

	tx, err := db.sqldb.Begin()
	if err != nil {
		return err
	}

	err = db.storeAccountOwner(ctx, accountOwner, tx)

	if err != nil {
		if rollbackErr := tx.Rollback(); rollbackErr != nil {
			return errors.Wrap(rollbackErr, err.Error()) // so we don't lose the original error
		}
		return err
	}
	if err = tx.Commit(); err != nil {
		return err
	}

	return nil
}

func (db *DB) storeAccountOwner(ctx context.Context, accountOwner domain.AccountOwner, tx *sql.Tx) error {

	sqlStatement := `
			INSERT INTO aws_account (account)
			VALUES ($1)
			ON CONFLICT DO NOTHING
			`
	var err error
	if _, err = tx.ExecContext(ctx, sqlStatement, accountOwner.AccountID); err != nil {
		return err
	}

	// Insert or update details of account owner
	if _, err = tx.ExecContext(ctx, insertPersonQuery, accountOwner.Owner.Login, accountOwner.Owner.Email, accountOwner.Owner.Name, accountOwner.Owner.Valid); err != nil {
		return err
	}

	sqlStatement = `SELECT id FROM person WHERE login=$1`
	row := tx.QueryRowContext(ctx, sqlStatement, accountOwner.Owner.Login)
	var personID int
	if err := row.Scan(&personID); err != nil {
		return err
	}

	sqlStatement = `SELECT id FROM aws_account WHERE account=$1`
	row = tx.QueryRowContext(ctx, sqlStatement, accountOwner.AccountID)
	var accountID int
	if err := row.Scan(&accountID); err != nil {
		return err
	}

	sqlStatement = `
			INSERT INTO account_owner
			VALUES ($1, $2)
			ON CONFLICT (aws_account_id)
			DO UPDATE SET person_id = $1, aws_account_id = $2
			`
	if _, err := tx.ExecContext(ctx, sqlStatement, personID, accountID); err != nil {
		return err
	}

	sqlStatement = `DELETE FROM account_champion WHERE aws_account_id=$1`
	if _, err := tx.ExecContext(ctx, sqlStatement, accountID); err != nil {
		return err
	}
	if len(accountOwner.Champions) > 0 {
		for _, person := range accountOwner.Champions {
			// Add champion to "person" table if champion does not exists in "person" table
			if _, err := tx.ExecContext(ctx, insertPersonQuery, person.Login, person.Email, person.Name, person.Valid); err != nil {
				return err
			}

			row := tx.QueryRowContext(ctx, `SELECT id FROM person WHERE login=$1`, person.Login)
			var champID int
			if err := row.Scan(&champID); err != nil {
				return err
			}
			sqlStatement = `
					INSERT INTO account_champion(person_id, aws_account_id)
					VALUES ($1, $2)
					ON CONFLICT DO NOTHING
					`
			if _, err := tx.ExecContext(ctx, sqlStatement, champID, accountID); err != nil {
				return err
			}
		}
	}
	return nil
}

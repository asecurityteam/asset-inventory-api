/*
JSON spec:
type CloudAssetDetails struct {
	PrivateIPAddresses []string          `json:"privateIpAddresses"`
	PublicIPAddresses  []string          `json:"publicIpAddresses"`
	Hostnames          []string          `json:"hostnames"`
	ResourceType       string            `json:"resourceType"`
	AccountID          string            `json:"accountId"`
	Region             string            `json:"region"`
	ARN                string            `json:"arn"`
	Tags               map[string]string `json:"tags"`
}
*/
select array_to_json(
               array_agg(
                       row_to_json(res_assigned_joined)
                   )
           )
from (
         select res_assigned.id,
                res_assigned.arn,
                res_assigned.private_ips as privateIpAddresses,
                res_assigned.public_ips as publicIpAddresses,
                res_assigned.hostnames,
                aws_account.account as accountId,
                aws_region.region,
                aws_resource_type.resource_type as resourceType,
                res_assigned.metadata
         from (select ar.id                            as id,
                      ar.arn_id                        as arn,
                      ar.meta                          as metadata,
                      ar.aws_account_id                as account_id,
                      ar.aws_region_id                 as region_id,
                      ar.aws_resource_type_id          as type_id,
                      array_agg(distinct private_ip)   as private_ips,
                      array_agg(distinct public_ip)    as public_ips,
                      array_agg(distinct aws_hostname) as hostnames
               from aws_resource ar
                        left join aws_private_ip_assignment pri
                                  on
                                      ar.id = pri.aws_resource_id
                        left join aws_public_ip_assignment pub
                                  on
                                      ar.id = pub.aws_resource_id
               where ar.id > :id_offset
                 and pri.not_before < :snapshot_timestamp
                 and (pri.not_after is null or pri.not_after > :snapshot_timestamp)
                 and pub.not_before < :snapshot_timestamp
                 and (pub.not_after is null or pub.not_after > :snapshot_timestamp)
               group by ar.id
               order by ar.id
               limit :page_size
              ) res_assigned
                  left join aws_account on res_assigned.account_id = aws_account.id
                  left join aws_region on res_assigned.region_id = aws_region.id
                  left join aws_resource_type on res_assigned.type_id = aws_resource_type.id
     ) res_assigned_joined;


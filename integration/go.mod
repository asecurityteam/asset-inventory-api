module github.com/asecurityteam/asset-inventory-api/integration

go 1.12

require (
	github.com/asecurityteam/asset-inventory-api/client v0.0.0
	github.com/stretchr/testify v1.3.0
)

replace github.com/asecurityteam/asset-inventory-api/client v0.0.0 => ../client

module github.com/lavalleeale/ContinuousIntegration/lib/auth

go 1.20

require (
	github.com/lavalleeale/ContinuousIntegration/lib/db v1.0.0
	gorm.io/gorm v1.25.1
)

require (
	github.com/google/uuid v1.3.0 // indirect
	github.com/jackc/pgpassfile v1.0.0 // indirect
	github.com/jackc/pgservicefile v0.0.0-20221227161230-091c0ba34f0a // indirect
	github.com/jackc/pgx/v5 v5.3.1 // indirect
	github.com/lib/pq v1.10.9 // indirect
	golang.org/x/text v0.10.0 // indirect
	gorm.io/driver/postgres v1.5.2 // indirect
)

require (
	github.com/jinzhu/inflection v1.0.0 // indirect
	github.com/jinzhu/now v1.1.5 // indirect
	golang.org/x/crypto v0.10.0
)

replace github.com/lavalleeale/ContinuousIntegration/lib/db => ../db

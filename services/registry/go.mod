module github.com/lavalleeale/ContinuousIntegration/services/registry

go 1.20

require (
	github.com/docker/distribution v2.8.2+incompatible
	github.com/docker/libtrust v0.0.0-20160708172513-aabc10ec26b7
	github.com/joho/godotenv v1.5.1
	github.com/lavalleeale/ContinuousIntegration/lib/auth v1.0.0
	github.com/lavalleeale/SessionSeal v0.0.0-20230616163435-35be5f28bfa5
)

replace github.com/lavalleeale/ContinuousIntegration/lib/auth => ../../lib/auth

replace github.com/lavalleeale/ContinuousIntegration/lib/db => ../../lib/db

require (
	github.com/google/uuid v1.3.0 // indirect
	github.com/gorilla/mux v1.8.0 // indirect
	github.com/jackc/pgpassfile v1.0.0 // indirect
	github.com/jackc/pgservicefile v0.0.0-20221227161230-091c0ba34f0a // indirect
	github.com/jackc/pgx/v5 v5.3.1 // indirect
	github.com/jinzhu/inflection v1.0.0 // indirect
	github.com/jinzhu/now v1.1.5 // indirect
	github.com/lavalleeale/ContinuousIntegration/lib/db v1.0.0 // indirect
	github.com/lib/pq v1.10.9 // indirect
	github.com/sirupsen/logrus v1.9.3 // indirect
	golang.org/x/crypto v0.10.0 // indirect
	golang.org/x/sys v0.9.0 // indirect
	golang.org/x/text v0.10.0 // indirect
	gorm.io/driver/postgres v1.5.2 // indirect
	gorm.io/gorm v1.25.1 // indirect
)

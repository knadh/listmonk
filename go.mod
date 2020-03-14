module github.com/knadh/listmonk

require (
	github.com/disintegration/imaging v1.6.2
	github.com/gofrs/uuid v3.2.0+incompatible
	github.com/jmoiron/sqlx v1.2.0
	github.com/jordan-wright/email v0.0.0-20200307200233-de844847de93
	github.com/knadh/goyesql/v2 v2.1.1
	github.com/knadh/koanf v0.8.1
	github.com/knadh/stuffbin v1.1.0
	github.com/labstack/echo v3.3.10+incompatible
	github.com/labstack/gommon v0.3.0 // indirect
	github.com/lib/pq v1.3.0
	github.com/niemeyer/pretty v0.0.0-20200227124842-a10e7caefd8e // indirect
	github.com/rhnvrm/simples3 v0.5.0
	github.com/spf13/pflag v1.0.5
	gopkg.in/check.v1 v1.0.0-20200227125254-8fa46927fb4f // indirect
	gopkg.in/volatiletech/null.v6 v6.0.0-20170828023728-0bef4e07ae1b
)

replace github.com/jordan-wright/email => github.com/knadh/email v0.0.0-20200206100304-6d2c7064c2e8

go 1.13

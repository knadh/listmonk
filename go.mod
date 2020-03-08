module github.com/knadh/listmonk

require (
	github.com/disintegration/imaging v1.5.0
	github.com/gofrs/uuid v3.2.0+incompatible
	github.com/jinzhu/gorm v1.9.1
	github.com/jmoiron/sqlx v1.2.0
	github.com/jordan-wright/email v0.0.0-20181027021455-480bedc4908b
	github.com/knadh/goyesql/v2 v2.1.1
	github.com/knadh/koanf v0.8.1
	github.com/knadh/stuffbin v1.0.0
	github.com/kr/pretty v0.1.0 // indirect
	github.com/labstack/echo v3.3.10+incompatible
	github.com/labstack/gommon v0.2.7 // indirect
	github.com/lib/pq v1.0.0
	github.com/mattn/go-colorable v0.0.9 // indirect
	github.com/mattn/go-isatty v0.0.4 // indirect
	github.com/rhnvrm/simples3 v0.5.0
	github.com/spf13/pflag v1.0.3
	github.com/valyala/bytebufferpool v1.0.0 // indirect
	github.com/valyala/fasttemplate v0.0.0-20170224212429-dcecefd839c4 // indirect
	golang.org/x/image v0.0.0-20181116024801-cd38e8056d9b // indirect
	gopkg.in/check.v1 v1.0.0-20180628173108-788fd7840127 // indirect
	gopkg.in/volatiletech/null.v6 v6.0.0-20170828023728-0bef4e07ae1b
)

replace github.com/jordan-wright/email => github.com/knadh/email v0.0.0-20200206100304-6d2c7064c2e8

go 1.13

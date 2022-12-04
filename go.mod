module github.com/slackhq/nebula

go 1.17

require (
	github.com/anmitsu/go-shlex v0.0.0-20200514113438-38f4b401e2be
	github.com/armon/go-radix v1.0.0
	github.com/cespare/xxhash/v2 v2.1.2 // indirect
	github.com/cyberdelia/go-metrics-graphite v0.0.0-20161219230853-39f87cc3b432
	github.com/flynn/noise v1.0.0
	github.com/gogo/protobuf v1.3.2
	github.com/golang/protobuf v1.5.2
	github.com/google/gopacket v1.1.19
	github.com/imdario/mergo v0.3.8
	github.com/jackpal/gateway v1.0.7
	github.com/kardianos/service v1.2.0
	github.com/miekg/dns v1.1.43
	github.com/nbrownus/go-metrics-prometheus v0.0.0-20210712211119-974a6260965f
	github.com/prometheus/client_golang v1.11.0
	github.com/prometheus/client_model v0.2.0 // indirect
	github.com/prometheus/procfs v0.7.3 // indirect
	github.com/rcrowley/go-metrics v0.0.0-20201227073835-cf1acfcdf475
	github.com/sirupsen/logrus v1.8.1
	github.com/skip2/go-qrcode v0.0.0-20200617195104-da1b6568686e
	github.com/songgao/water v0.0.0-20200317203138-2b4b6d7c09d8
	github.com/stretchr/testify v1.7.0
	github.com/vishvananda/netlink v1.1.0
	github.com/vishvananda/netns v0.0.0-20211101163701-50045581ed74 // indirect
	golang.org/x/crypto v0.0.0-20211202192323-5770296d904e
	golang.org/x/net v0.0.0-20211112202133-69e39bad7dc2
	golang.org/x/sys v0.0.0-20211103235746-7861aae1554b
	golang.zx2c4.com/wintun v0.0.0-20211104114900-415007cec224
	golang.zx2c4.com/wireguard/windows v0.5.1
	google.golang.org/protobuf v1.27.1
	gopkg.in/yaml.v2 v2.4.0
)

require (
	fyne.io/fyne v1.4.3 // indirect
	fyne.io/fyne/v2 v2.1.2 // indirect
	github.com/beorn7/perks v1.0.1 // indirect
	github.com/davecgh/go-spew v1.1.1 // indirect
	github.com/denisbrodbeck/machineid v1.0.1 // indirect
	github.com/fredbi/uri v0.0.0-20181227131451-3dcfdacbaaf3 // indirect
	github.com/fsnotify/fsnotify v1.4.9 // indirect
	github.com/fyne-io/mobile v0.1.2 // indirect
	github.com/go-gl/gl v0.0.0-20210813123233-e4099ee2221f // indirect
	github.com/go-gl/glfw/v3.3/glfw v0.0.0-20211024062804-40e447a793be // indirect
	github.com/go-sql-driver/mysql v1.6.0 // indirect
	github.com/godbus/dbus/v5 v5.0.4 // indirect
	github.com/goki/freetype v0.0.0-20181231101311-fa8a33aabaff // indirect
	github.com/jstemmer/gotags v1.4.1 // indirect
	github.com/lestrrat-go/file-rotatelogs v2.4.0+incompatible // indirect
	github.com/lestrrat-go/strftime v1.0.5 // indirect
	github.com/lestrrat/go-file-rotatelogs v0.0.0-20180223000712-d3151e2a480f // indirect
	github.com/lestrrat/go-strftime v0.0.0-20180220042222-ba3bf9c1d042 // indirect
	github.com/matttproud/golang_protobuf_extensions v1.0.1 // indirect
	github.com/pkg/errors v0.9.1 // indirect
	github.com/pmezard/go-difflib v1.0.0 // indirect
	github.com/prometheus/common v0.32.1 // indirect
	github.com/srwiley/oksvg v0.0.0-20200311192757-870daf9aa564 // indirect
	github.com/srwiley/rasterx v0.0.0-20200120212402-85cb7272f5e9 // indirect
	github.com/yuin/goldmark v1.4.1 // indirect
	golang.org/x/image v0.0.0-20200430140353-33d19683fad8 // indirect
	golang.org/x/mod v0.5.1 // indirect
	golang.org/x/term v0.0.0-20210927222741-03fcf44c2211 // indirect
	golang.org/x/text v0.3.8-0.20211004125949-5bd84dd9b33b // indirect
	golang.org/x/tools v0.1.9 // indirect
	golang.org/x/xerrors v0.0.0-20200804184101-5ec99f83aff1 // indirect
	gopkg.in/yaml.v3 v3.0.0-20200313102051-9f266ea9e77c // indirect
)

require screen v1.0.0

replace screen v1.0.0 => ./screen

require messages v1.0.0

replace messages v1.0.0 => ./messages

require nh_util v1.0.0

replace nh_util v1.0.0 => ./nh_util

require platform v1.0.0

replace platform v1.0.0 => ./platform

require router v1.0.0

replace router v1.0.0 => ./router

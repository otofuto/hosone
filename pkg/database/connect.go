package database

import (
	"crypto/tls"
	"crypto/x509"
	"database/sql"
	"errors"
	"math"
	"os"
	"strings"

	"github.com/go-sql-driver/mysql"
)

func Connect() *sql.DB {
	connectionstring := os.Getenv("DB_USER") + ":" + os.Getenv("DB_PASS") + "@" + os.Getenv("DB_HOST") + "/" + os.Getenv("DB_NAME")
	db, err := sql.Open("mysql", connectionstring)
	if err != nil {
		panic(err.Error())
	}
	//log.Println("db connect")

	return db
}

func registerTlsConfig(pemPath, tlsConfigKey string) error {
	caCertPool := x509.NewCertPool()
	pem, err := os.ReadFile(pemPath)
	if err != nil {
		return err
	}

	if ok := caCertPool.AppendCertsFromPEM(pem); !ok {
		return errors.New("Failed to append PEM.")
	}
	mysql.RegisterTLSConfig(tlsConfigKey, &tls.Config{
		ClientCAs:          caCertPool,
		InsecureSkipVerify: true,
	})

	return err
}

func Escape(str string) string {
	ret := strings.Replace(str, "\\", "\\\\", -1)
	ret = strings.Replace(ret, "\"", "\\\"", -1)
	ret = strings.Replace(ret, "'", "\\'", -1)
	ret = strings.Replace(ret, "\t", "\\t", -1)
	ret = strings.Replace(ret, "\r", "\\r", -1)
	ret = strings.Replace(ret, "\n", "\\n", -1)

	return ret
}

func Int64ToInt(i int64) int {
	if i < math.MinInt32 || i > math.MaxInt32 {
		return 0
	} else {
		return int(i)
	}
}

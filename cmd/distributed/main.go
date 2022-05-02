package main

import (
	"context"
	"crypto/tls"
	"crypto/x509"
	"database/sql"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"net"
	"net/http"
	"os"
	"os/signal"
	"path/filepath"
	"regexp"
	"strings"
	"time"

	"github.com/bluebrown/sqlite-bug/pkg/models"
	"github.com/canonical/go-dqlite/app"
	"github.com/canonical/go-dqlite/client"
	"github.com/golang-migrate/migrate/v4"
	"github.com/golang-migrate/migrate/v4/database/sqlite3"
	_ "github.com/golang-migrate/migrate/v4/source/file"
	"github.com/julienschmidt/httprouter"
	"github.com/volatiletech/sqlboiler/v4/boil"
	"github.com/volatiletech/sqlboiler/v4/queries"
	"golang.org/x/sys/unix"
)

var (
	dbName   = "test"
	dataDir  = "./data"
	sqlPort  = "9000"
	httpPort = "8080"
)

func main() {
	boil.DebugMode = true
	boil.DebugWriter = os.Stderr

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	if err := run(ctx); err != nil {
		cancel()
		log.Printf("error: %v", err)
		time.Sleep(2 * time.Second)
		os.Exit(1)
	}
}

func run(ctx context.Context) error {
	certPath, ok := os.LookupEnv("CERT_PATH")
	if !ok {
		return fmt.Errorf("CERT_PATH not set")
	}

	cert, err := tls.LoadX509KeyPair(filepath.Join(certPath, "tls.crt"), filepath.Join(certPath, "tls.key"))
	if err != nil {
		return err
	}
	data, err := ioutil.ReadFile(filepath.Join(certPath, "tls.crt"))
	if err != nil {
		return err
	}
	pool := x509.NewCertPool()
	pool.AppendCertsFromPEM(data)

	podDns, statefulZeroDns, isZero := clusterDns()
	var cluster []string
	if !isZero {
		cluster = []string{net.JoinHostPort(statefulZeroDns, sqlPort)}
	}

	log.Printf("pod dns: %s, cluster %v", podDns, cluster)

	dqlite, err := app.New(
		dataDir,
		app.WithTLS(app.SimpleTLSConfig(cert, pool)),
		app.WithAddress(net.JoinHostPort(podDns, sqlPort)),
		app.WithCluster(cluster),
		app.WithLogFunc(func(l client.LogLevel, format string, a ...interface{}) {
			if l < 2 {
				return
			}
			log.Printf(fmt.Sprintf("%s: %s\n", l.String(), format), a...)
		}),
	)
	if err != nil {
		return err
	}

	// this blocks until a connection to the leader
	// is made, so the server further down won't start
	if err := dqlite.Ready(ctx); err != nil {
		return err
	}

	db, err := dqlite.Open(ctx, dbName)
	if err != nil {
		return err
	}

	defer func() {
		log.Println("closing dqlite resources")
		if err := db.Close(); err != nil {
			log.Printf("error closing db: %v", err)
		}
		if err := dqlite.Handover(ctx); err != nil {
			log.Printf("error: %v", err)
		}
		if err := dqlite.Close(); err != nil {
			log.Printf("error: %v", err)
		}
	}()

	if err := applyMigrations(ctx, db, dbName); err != nil {
		return err
	}

	return runGracefully(&http.Server{
		Addr:    "0.0.0.0:" + httpPort,
		Handler: newRouter(db),
	})

}

func clusterDns() (podDns, statefulZeroDns string, isZero bool) {
	dnsParts := []string{"POD_NAME", "SERVICE_NAME", "NAMESPACE", "CLUSTER_SUFFIX"}
	for i, d := range dnsParts {
		if v, ok := os.LookupEnv(d); ok {
			dnsParts[i] = v
		} else {
			panic(fmt.Sprintf("%s not set", d))
		}
	}
	isZero = strings.HasSuffix(dnsParts[0], "-0")
	podDns = strings.Join(dnsParts, ".")
	dnsParts[0] = regexp.MustCompile(`-\d+$`).ReplaceAllString(dnsParts[0], "-0")
	statefulZeroDns = strings.Join(dnsParts, ".")
	return podDns, statefulZeroDns, isZero
}

func runGracefully(server *http.Server) error {
	sigs := make(chan os.Signal, 1)
	signal.Notify(sigs, unix.SIGPWR, unix.SIGINT, unix.SIGQUIT, unix.SIGTERM)
	go func() {
		log.Println("starting server")
		if err := server.ListenAndServe(); err != http.ErrServerClosed {
			log.Printf("error: %v", err)
		}
	}()
	<-sigs
	log.Println("shutting down")
	timeout, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	return server.Shutdown(timeout)
}

func applyMigrations(ctx context.Context, db *sql.DB, dbName string) error {
	driver, err := sqlite3.WithInstance(db, &sqlite3.Config{})
	if err != nil {
		return err
	}
	m, err := migrate.NewWithDatabaseInstance("file://./sql/migrations", dbName, driver)
	if err != nil {
		return err
	}
	m.Up()
	return nil
}

func newRouter(db *sql.DB) http.Handler {
	router := httprouter.New()

	// ping is used for health checks
	router.GET("/ping", func(w http.ResponseWriter, r *http.Request, _ httprouter.Params) {
		w.Write([]byte("pong"))
	})

	router.POST("/debug", func(w http.ResponseWriter, r *http.Request, _ httprouter.Params) {
		var debug models.Debug
		if err := json.NewDecoder(r.Body).Decode(&debug); err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}

		if err := debug.Insert(r.Context(), db, boil.Infer()); err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		w.Header().Add("Content-Type", "application/json")
		w.Header().Add("Content-Location", fmt.Sprintf("/debug/%d", debug.ID))
		w.WriteHeader(http.StatusCreated)
		if err := json.NewEncoder(w).Encode(debug); err != nil {
			log.Printf("error: %v", err)
		}
	})

	router.GET("/inc", func(w http.ResponseWriter, r *http.Request, _ httprouter.Params) {
		log.Println("incrementing counter")
		if _, err := queries.Raw("update counter set count = count + 1 where id = 1").ExecContext(r.Context(), db); err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
	})

	router.GET("/", func(w http.ResponseWriter, r *http.Request, _ httprouter.Params) {
		log.Println("serving counter")
		var count int64
		if err := queries.Raw("select count from counter where id = 1").QueryRowContext(r.Context(), db).Scan(&count); err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		fmt.Fprintf(w, "count: %d\n", count)
	})

	return router
}

package store

import (
	"context"
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"runtime/debug"
	"time"

	"github.com/mjl-/bstore"

	"github.com/mjl-/mox/metrics"
	"github.com/mjl-/mox/mlog"
	"github.com/mjl-/mox/mox-"
	"github.com/mjl-/mox/moxvar"
)

// AccountRemove represents the scheduled removal of an account, when its last
// reference goes away.
type AccountRemove struct {
	AccountName string
}

// AuthDB and AuthDBTypes are exported for ../backup.go.
var AuthDB *bstore.DB
var AuthDBTypes = []any{TLSPublicKey{}, LoginAttempt{}, LoginAttemptState{}, AccountRemove{}}

var loginAttemptCleanerStop chan chan struct{}

// Init opens auth.db and starts the login writer.
func Init(ctx context.Context) error {
	if AuthDB != nil {
		return fmt.Errorf("already initialized")
	}
	pkglog := mlog.New("store", nil)
	p := mox.DataDirPath("auth.db")
	os.MkdirAll(filepath.Dir(p), 0770)
	opts := bstore.Options{Timeout: 5 * time.Second, Perm: 0660, RegisterLogger: moxvar.RegisterLogger(p, pkglog.Logger)}
	var err error
	AuthDB, err = bstore.Open(ctx, p, &opts, AuthDBTypes...)
	if err != nil {
		return err
	}

	// List pending account removals, and process them one by one, committing each
	// individually.
	removals, err := bstore.QueryDB[AccountRemove](ctx, AuthDB).List()
	if err != nil {
		return fmt.Errorf("listing scheduled account removals: %v", err)
	}
	for _, removal := range removals {
		if err := removeAccount(pkglog, removal.AccountName); err != nil {
			pkglog.Errorx("removing old account", err, slog.String("account", removal.AccountName))
		}
	}

	startLoginAttemptWriter()
	loginAttemptCleanerStop = make(chan chan struct{})

	go func() {
		defer func() {
			x := recover()
			if x == nil {
				return
			}

			mlog.New("store", nil).Error("unhandled panic in LoginAttemptCleanup", slog.Any("err", x))
			debug.PrintStack()
			metrics.PanicInc(metrics.Store)

		}()

		t := time.NewTicker(24 * time.Hour)
		for {
			err := LoginAttemptCleanup(ctx)
			pkglog.Check(err, "cleaning up old historic login attempts")

			select {
			case c := <-loginAttemptCleanerStop:
				c <- struct{}{}
				return
			case <-t.C:
			case <-ctx.Done():
				return
			}
		}
	}()

	// Initialize NATS client if configured
	if err := InitNATS(pkglog, mox.Conf.Static.NATS); err != nil {
		pkglog.Errorx("initializing NATS client", err)
		// Don't fail startup if NATS initialization fails, just log the error
	}

	return nil
}

// Close closes auth.db and stops the login writer.
func Close() error {
	if AuthDB == nil {
		return fmt.Errorf("not open")
	}

	stopc := make(chan struct{})
	writeLoginAttemptStop <- stopc
	<-stopc

	stopc = make(chan struct{})
	loginAttemptCleanerStop <- stopc
	<-stopc

	// Close NATS client if it exists
	if natsClient := GetNATSClient(); natsClient != nil {
		if err := natsClient.Close(); err != nil {
			pkglog := mlog.New("store", nil)
			pkglog.Errorx("closing NATS client", err)
		}
	}

	err := AuthDB.Close()
	AuthDB = nil

	return err
}

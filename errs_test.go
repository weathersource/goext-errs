// Copyright 2021 Airbus Defence and Space
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package errs_test

import (
	"context"
	"fmt"
	"net"
	"syscall"
	"testing"
	"time"

	"github.com/airbusgeo/errs"
	"github.com/pkg/errors"
	oldcontext "golang.org/x/net/context"
	"google.golang.org/api/googleapi"
)

func checkCause(t *testing.T, err error) {
	errorcause := errors.Cause(err)
	if errorcause.Error() != "initial error" {
		t.Errorf("wrapped error cause returns [%s] instead of [initial error]", errorcause.Error())
	}
}
func TestTempErr(t *testing.T) {
	err := fmt.Errorf("initial error")
	if errs.Temporary(err) {
		t.Error("expected plain error to be non temporary")
	}
	checkCause(t, err)

	pkgwrappederr := errors.Wrap(err, "pkg wrap 1")
	if errs.Temporary(pkgwrappederr) {
		t.Error("expected pkg wrapped plain error to be non temporary")
	}
	checkCause(t, pkgwrappederr)

	temperror := errs.MakeTemporary(err)
	if !errs.Temporary(temperror) {
		t.Error("expected temperr.Wrap'd error to be temporary")
	}
	checkCause(t, temperror)

	pkgwrappedtemperror := errors.Wrap(temperror, "pkg wrap 2")
	if !errs.Temporary(pkgwrappedtemperror) {
		t.Error("expected pkg wrapped temporary error to be temporary")
	}
	checkCause(t, pkgwrappedtemperror)

	permerror := errs.MakePermanent(err)
	if errs.Temporary(permerror) {
		t.Error("expected permanent error to be permanent, duh")
	}
	checkCause(t, permerror)

	permtemperror := errs.MakePermanent(temperror)
	if errs.Temporary(permtemperror) {
		t.Error("expected explicitely marked permanent (previously temp) error to be permanent")
	}
	checkCause(t, permtemperror)

	permwrappedtemperror := errs.MakePermanent(pkgwrappedtemperror)
	if errs.Temporary(permwrappedtemperror) {
		t.Error("expected explicitely marked permanent wrapped (previously temp) error to be permanent")
	}
	checkCause(t, permwrappedtemperror)

	wrappedpermtemperror := errors.Wrap(permtemperror, "pkg wrap 3")
	if errs.Temporary(wrappedpermtemperror) {
		t.Error("expected wrapped permtemp error to be permanent")
	}
	checkCause(t, wrappedpermtemperror)

	if errs.Temporary(nil) {
		t.Error("nil error is not temporary")
	}

}

type temperr struct {
	error
	t bool
}

func (t temperr) Temporary() bool {
	return t.t
}

func wrap(err error) error {
	return fmt.Errorf("wrapped %w", err)
}

func cancelTest(ctx context.Context) error {
	<-ctx.Done()
	return ctx.Err()
}

func oldCancelTest(ctx oldcontext.Context) error {
	<-ctx.Done()
	return ctx.Err()
}

func TestTemporary(t *testing.T) {
	err := temperr{errors.New("temp"), true}
	werr := wrap(err)
	if !errs.Temporary(err) || !errs.Temporary(werr) {
		t.Error("temp")
	}
	err.t = false
	werr = wrap(err)
	if errs.Temporary(err) || errs.Temporary(werr) {
		t.Error("temp")
	}
	ctx, _ := context.WithTimeout(context.Background(), 50*time.Millisecond)
	cerr := cancelTest(ctx)
	werr = wrap(err)
	if !errs.Temporary(cerr) || errs.Temporary(werr) {
		t.Error("temp")
	}
	ctx, cncl := context.WithCancel(context.Background())
	go func() {
		time.Sleep(50 * time.Millisecond)
		cncl()
	}()
	cerr = cancelTest(ctx)
	werr = wrap(err)
	if !errs.Temporary(cerr) || errs.Temporary(werr) {
		t.Error("temp")
	}
}
func TestOldContext(t *testing.T) {
	ctx, _ := oldcontext.WithTimeout(oldcontext.Background(), 50*time.Millisecond)
	cerr := oldCancelTest(ctx)
	werr := wrap(cerr)
	if !errs.Temporary(cerr) || !errs.Temporary(werr) {
		t.Error("temp")
	}
	ctx, cncl := oldcontext.WithCancel(oldcontext.Background())
	go func() {
		time.Sleep(50 * time.Millisecond)
		cncl()
	}()
	cerr = oldCancelTest(ctx)
	werr = wrap(cerr)
	if !errs.Temporary(cerr) || !errs.Temporary(werr) {
		t.Error("temp")
	}
}
func TestNil(t *testing.T) {
	if errs.Temporary(nil) {
		t.Error("nil is temp")
	}
}
func TestErrno(t *testing.T) {
	err := syscall.EIO
	if !errs.Temporary(err) {
		t.Error("eio not temp")
	}
	if !errs.Temporary(wrap(err)) {
		t.Error("wrapped eio not temp")
	}
	err = syscall.EAGAIN
	if !errs.Temporary(err) {
		t.Error("eagain not temp")
	}
	if !errs.Temporary(wrap(err)) {
		t.Error("wrapped eagain not temp")
	}
}

func TestGoogleApiError(t *testing.T) {
	gerr := &googleapi.Error{Code: 400}
	if errs.Temporary(gerr) {
		t.Error("400 is temp")
	}
	gerr = &googleapi.Error{Code: 429}
	if !errs.Temporary(gerr) {
		t.Error("429 is  not temp")
	}
	gerr = &googleapi.Error{Code: 500}
	if !errs.Temporary(gerr) {
		t.Error("500 is  not temp")
	}
}

func TestDialTimeout(t *testing.T) {
	t.Skip() //this test only works if port 22222 is firewalled/dropped
	l, _ := net.ListenTCP("tcp4", &net.TCPAddr{
		IP:   net.IPv4(127, 0, 0, 1),
		Port: 22222,
	})
	defer l.Close()

	d := net.Dialer{Timeout: 50 * time.Millisecond}
	_, err := d.Dial("tcp", "127.0.0.1:22222")
	if err != nil {
		t.Error(err)
	}
	if !errs.Temporary(err) {
		t.Error("not temporary")
	}

}

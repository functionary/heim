package console

import (
	"testing"

	"heim/backend/mock"
	"heim/proto/security"

	"golang.org/x/net/context"

	. "github.com/smartystreets/goconvey/convey"
)

func TestSetRoomPasscode(t *testing.T) {
	ctx := context.Background()
	kms := security.LocalKMS()
	kms.SetMasterKey(make([]byte, security.AES256.KeySize()))

	Convey("Room not given", t, func() {
		term := &testTerm{}
		runCommand(&Controller{}, "set-room-passcode", term, nil)
		So(term.String(), ShouldStartWith, "error: invalid command")
	})

	Convey("Room not locked", t, func() {
		ctrl := &Controller{
			backend: &mock.TestBackend{},
			kms:     kms,
		}
		term := &testTerm{}
		runCommand(ctrl, "set-room-passcode", term, []string{"test"})
		So(term.String(), ShouldEqual, "error: room doesn't exist or isn't locked\r\n")
	})

	Convey("Set passcode on room", t, func() {
		ctrl := &Controller{
			backend: &mock.TestBackend{},
			kms:     kms,
		}

		term := &testTerm{}
		runCommand(ctrl, "lock-room", term, []string{"test"})
		So(term.String(), ShouldStartWith, "Room test locked with new key")

		room, err := ctrl.backend.GetRoom("test")
		So(err, ShouldBeNil)
		mkey, err := room.GenerateMasterKey(ctx, kms)
		So(err, ShouldBeNil)

		term = &testTerm{password: "hunter2"}
		runCommand(ctrl, "set-room-passcode", term, []string{"test"})
		So(term.String(), ShouldStartWith, "Passcode added to test: ")

		capabilityID, err := security.GetCapabilityIDForPasscode(mkey.Nonce(), []byte("hunter2"))
		So(err, ShouldBeNil)

		capability, err := room.GetCapability(ctx, capabilityID)
		So(err, ShouldBeNil)
		So(capability, ShouldNotBeNil)
	})
}

func TestLockRoom(t *testing.T) {
	ctx := context.Background()
	kms := security.LocalKMS()
	kms.SetMasterKey(make([]byte, security.AES256.KeySize()))

	Convey("Usage and flags", t, func() {
		term := &testTerm{}
		runCommand(&Controller{}, "lock-room", term, []string{"-h"})
		So(term.String(), ShouldStartWith, "Usage of lock-room:")
	})

	Convey("Room not given", t, func() {
		term := &testTerm{}
		runCommand(&Controller{}, "lock-room", term, nil)
		So(term.String(), ShouldStartWith, "error: room name must be given\r\n")
	})

	// mock doesn't do not found
	SkipConvey("Room not found", t, func() {
		ctrl := &Controller{
			backend: &mock.TestBackend{},
			kms:     kms,
		}
		term := &testTerm{}
		runCommand(ctrl, "lock-room", term, []string{"!!!!"})
		So(term.String(), ShouldStartWith, "error: room name must be given\r\n")
	})

	Convey("Locking room that is already locked", t, func() {
		ctrl := &Controller{
			backend: &mock.TestBackend{},
			kms:     kms,
		}

		room, err := ctrl.backend.GetRoom("test")
		So(err, ShouldBeNil)
		orig, err := room.GenerateMasterKey(ctx, kms)
		So(err, ShouldBeNil)

		Convey("Requires --force", func() {
			term := &testTerm{}
			runCommand(ctrl, "lock-room", term, []string{"test"})
			So(term.String(), ShouldEqual,
				"error: room already locked; use --force to relock and invalidate all previous grants\r\n")
		})

		Convey("Proceeds with --force", func() {
			_ = orig
			term := &testTerm{}
			runCommand(ctrl, "lock-room", term, []string{"--force", "test"})
			So(term.String(), ShouldStartWith,
				"Overwriting existing key.\r\nRoom test locked with new key")
			rk, err := room.MasterKey(ctx)
			So(err, ShouldBeNil)
			So(rk, ShouldNotResemble, orig)
		})
	})
}
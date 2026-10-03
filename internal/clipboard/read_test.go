package clipboard

import (
	"errors"
	"strings"
	"testing"
)

func TestReadApprovalCancellationAndDocumentLifetime(t *testing.T) {
	for _, stage := range []string{"success", "cancel", "confirm-error", "before-close", "during-close", "navigate", "read-error", "after-read", "oversized"} {
		t.Run(stage, func(t *testing.T) {
			var queued func()
			active, generation, confirmations, reads, replies := true, uint64(1), 0, 0, 0
			r := &Reader{
				Dispatch: func(fn func()) { queued = fn }, Active: func() bool { return active }, Generation: func() uint64 { return generation },
				Confirm: func() (bool, error) {
					confirmations++
					if stage == "during-close" {
						active = false
					}
					if stage == "navigate" {
						generation++
					}
					if stage == "confirm-error" {
						return false, ErrNative
					}
					return stage != "cancel", nil
				},
				Read: func() (string, error) {
					reads++
					if stage == "read-error" {
						return "private", ErrBusy
					}
					if stage == "after-read" {
						generation++
					}
					if stage == "oversized" {
						return strings.Repeat("x", MaxTextBytes+1), nil
					}
					return "\ud55c\uae00\n\U0001f642", nil
				},
			}
			var result ReadResult
			var resultErr error
			if err := r.ReadText(func(value ReadResult, err error) { replies++; result, resultErr = value, err }); err != nil {
				t.Fatal(err)
			}
			if confirmations != 0 || reads != 0 || replies != 0 {
				t.Fatal("read did not defer")
			}
			if err := r.ReadText(func(ReadResult, error) {}); !errors.Is(err, ErrPending) {
				t.Fatal("duplicate read queued")
			}
			if stage == "before-close" {
				active = false
			}
			queued()
			if replies != 1 || r.pending.Load() {
				t.Fatal("completion or pending cleanup failed")
			}
			if stage == "success" {
				if resultErr != nil || result.Text == nil || *result.Text != "\ud55c\uae00\n\U0001f642" {
					t.Fatal(result, resultErr)
				}
			} else if result.Text != nil {
				t.Fatal("text returned on cancellation or failure")
			}
			if stage == "cancel" && (!result.Cancelled || resultErr != nil) {
				t.Fatal("cancellation lost")
			}
			if stage != "success" && stage != "cancel" && resultErr == nil {
				t.Fatal("failure accepted")
			}
			if stage == "cancel" || stage == "confirm-error" || stage == "before-close" || stage == "during-close" || stage == "navigate" {
				if reads != 0 {
					t.Fatal("clipboard accessed without current approval")
				}
			}
		})
	}
}

func TestReadMissingCallbacksCannotAccessClipboard(t *testing.T) {
	if err := (&Reader{}).ReadText(func(ReadResult, error) {}); !errors.Is(err, ErrInactive) {
		t.Fatal(err)
	}
}

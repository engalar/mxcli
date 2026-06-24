// SPDX-License-Identifier: Apache-2.0

//go:build !linux

package goldenfs

import "errors"

// ErrNotSupported is returned on non-Linux platforms where FUSE is unavailable.
var ErrNotSupported = errors.New("goldenfs: FUSE overlay requires Linux")

func Open(_ string, _ ...Option) (Committer, error) { return nil, ErrNotSupported }

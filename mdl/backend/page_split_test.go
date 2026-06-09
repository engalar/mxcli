// SPDX-License-Identifier: Apache-2.0
package backend_test

import (
	"testing"

	"github.com/mendixlabs/mxcli/mdl/backend"
	"github.com/mendixlabs/mxcli/mdl/backend/mock"
	mprbackend "github.com/mendixlabs/mxcli/mdl/backend/mpr"
)

// TestPageBackendSplit 是编译期断言——测试通过即证明接口满足正确。
func TestPageBackendSplit(t *testing.T) {
	var _ backend.PageReader = (*mprbackend.MprBackend)(nil)
	var _ backend.PageWriter = (*mprbackend.MprBackend)(nil)
	var _ backend.PageBackend = (*mprbackend.MprBackend)(nil)

	var _ backend.PageReader = (*mock.MockBackend)(nil)
	var _ backend.PageWriter = (*mock.MockBackend)(nil)
	var _ backend.PageBackend = (*mock.MockBackend)(nil)

	t.Log("PageReader + PageWriter + PageBackend all satisfied by MprBackend and MockBackend")
}

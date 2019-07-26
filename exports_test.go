/*
 * Copyright 2018 The NATS Authors
 * Licensed under the Apache License, Version 2.0 (the "License");
 * you may not use this file except in compliance with the License.
 * You may obtain a copy of the License at
 *
 * http://www.apache.org/licenses/LICENSE-2.0
 *
 * Unless required by applicable law or agreed to in writing, software
 * distributed under the License is distributed on an "AS IS" BASIS,
 * WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 * See the License for the specific language governing permissions and
 * limitations under the License.
 */

package jwt

import (
	"testing"
	"time"
)

func TestSimpleExportValidation(t *testing.T) {
	e := &Export{Subject: "foo", Type: Stream}

	vr := CreateValidationResults()
	e.Validate(vr)

	if !vr.IsEmpty() {
		t.Errorf("simple export should validate cleanly")
	}

	e.Type = Stream
	vr = CreateValidationResults()
	e.Validate(vr)

	if !vr.IsEmpty() {
		t.Errorf("simple export should validate cleanly")
	}
}

func TestInvalidExportType(t *testing.T) {
	i := &Export{Subject: "foo", Type: Unknown}

	vr := CreateValidationResults()
	i.Validate(vr)

	if vr.IsEmpty() {
		t.Errorf("export with bad type should not validate cleanly")
	}

	if !vr.IsBlocking(true) {
		t.Errorf("invalid type is blocking")
	}
}

func TestOverlappingExports(t *testing.T) {
	i := &Export{Subject: "bar.foo", Type: Stream}
	i2 := &Export{Subject: "bar.*", Type: Stream}

	exports := &Exports{}
	exports.Add(i, i2)

	vr := CreateValidationResults()
	exports.Validate(vr)

	if len(vr.Issues) != 1 {
		t.Errorf("export has overlapping subjects")
	}
}

func TestDifferentExportTypes_OverlapOK(t *testing.T) {
	i := &Export{Subject: "bar.foo", Type: Service}
	i2 := &Export{Subject: "bar.*", Type: Stream}

	exports := &Exports{}
	exports.Add(i, i2)

	vr := CreateValidationResults()
	exports.Validate(vr)

	if len(vr.Issues) != 0 {
		t.Errorf("should allow overlaps on different export kind")
	}
}

func TestDifferentExportTypes_SameSubjectOK(t *testing.T) {
	i := &Export{Subject: "bar", Type: Service}
	i2 := &Export{Subject: "bar", Type: Stream}

	exports := &Exports{}
	exports.Add(i, i2)

	vr := CreateValidationResults()
	exports.Validate(vr)

	if len(vr.Issues) != 0 {
		t.Errorf("should allow overlaps on different export kind")
	}
}

func TestSameExportType_SameSubject(t *testing.T) {
	i := &Export{Subject: "bar", Type: Service}
	i2 := &Export{Subject: "bar", Type: Service}

	exports := &Exports{}
	exports.Add(i, i2)

	vr := CreateValidationResults()
	exports.Validate(vr)

	if len(vr.Issues) != 1 {
		t.Errorf("should not allow same subject on same export kind")
	}
}

func TestExportRevocation(t *testing.T) {
	akp := createAccountNKey(t)
	apk := publicKey(akp, t)
	account := NewAccountClaims(apk)
	e := &Export{Subject: "foo", Type: Stream}

	account.Exports.Add(e)

	pubKey := "bar"
	now := time.Now()

	// test that clear is safe before we add any
	e.ClearRevocation(pubKey)

	if e.IsRevokedAt(pubKey, now) {
		t.Errorf("no revocation was added so is revoked should be false")
	}

	e.RevokeAt(pubKey, now.Add(time.Second*100))

	if !e.IsRevokedAt(pubKey, now) {
		t.Errorf("revocation should hold when timestamp is in the future")
	}

	if e.IsRevokedAt(pubKey, now.Add(time.Second*150)) {
		t.Errorf("revocation should time out")
	}

	e.RevokeAt(pubKey, now.Add(time.Second*50)) // shouldn't change the revocation, you can't move it in

	if !e.IsRevokedAt(pubKey, now.Add(time.Second*60)) {
		t.Errorf("revocation should hold, 100 > 50")
	}

	encoded, _ := account.Encode(akp)
	decoded, _ := DecodeAccountClaims(encoded)

	if !decoded.Exports[0].IsRevokedAt(pubKey, now.Add(time.Second*60)) {
		t.Errorf("revocation should last across encoding")
	}

	e.ClearRevocation(pubKey)

	if e.IsRevokedAt(pubKey, now) {
		t.Errorf("revocations should be cleared")
	}

	e.RevokeAt(pubKey, now.Add(time.Second*1000))

	if !e.IsRevoked(pubKey) {
		t.Errorf("revocation be true we revoked in the future")
	}
}

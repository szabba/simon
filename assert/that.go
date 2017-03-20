//   This Source Code Form is subject to the terms of the Mozilla Public
//   License, v. 2.0. If a copy of the MPL was not distributed with this
//   file, You can obtain one at http://mozilla.org/MPL/2.0/.

package assert

type ErrorFunc func(msg string, args ...interface{})

func That(cond bool, onError ErrorFunc, msg string, args ...interface{}) {
	if cond {
		return
	}
	onError(msg, args...)
}

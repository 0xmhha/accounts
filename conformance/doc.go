// Package conformance holds language-neutral golden vectors (vectors/*.json)
// and a runner (runner_test.go) that asserts the SDK reproduces them. The
// vectors are the cross-implementation contract: any SDK, in any language, must
// reproduce the same outputs.
package conformance

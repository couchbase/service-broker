// Package test provides end-to-end testing of the service broker.
//
// Testing uses the native go testing framework.  When any test is run a good
// configuration is used to start the service broker (see TestMain for default
// parameters).  Tests are run end-to-end using the local service broker API.
//
// The service broker has no dependencies against a live Kubernetes cluster when
// tested.  We instead use a fake client layer, populate with fixtures and then
// validate against this.
//
// Tests should only verify one thing and should aim to fail fast.  Tests are
// orgnaized into domain specific files as follows:
//
// api_test - Global API related functionality e.g. security and content type
//            processing.
package test

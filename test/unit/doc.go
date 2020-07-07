// Copyright 2020 Couchbase, Inc.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file  except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the  License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

// Package unit_test provides end-to-end testing of the service broker.
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
package unit

<a id="markdown-Asset Inventory API" name="Asset-Inventory-API"></a>
# Asset Inventory API - An API service to store and retrieve asset inventory data
[![Documentation](https://godoc.org/github.com/asecurityteam/asset-inventory-api?status.svg)](http://godoc.org/github.com/asecurityteam/asset-inventory-api)
[![Build Status](https://travis-ci.org/asecurityteam/asset-inventory-api.png?branch=master)](https://travis-ci.org/asecurityteam/asset-inventory-api)
[![codecov.io](https://codecov.io/github/asecurityteam/asset-inventory-api/coverage.svg?branch=master)](https://codecov.io/github/asecurityteam/asset-inventory-api?branch=master)

<https://github.com/asecurityteam/asset-inventory-api>

<!-- TOC -->

- [Asset Inventory API](#Asset-Inventory-API)
    - [Overview](#overview)
    - [Quick Start](#quick-start)
    - [Configuration](#configuration)
    - [Status](#status)
    - [Contributing](#contributing)
        - [Building And Testing](#building-and-testing)
        - [Quality Gates](#quality-gates)
        - [License](#license)
        - [Contributing Agreement](#contributing-agreement)

<!-- /TOC -->

<a id="markdown-overview" name="overview"></a>
## Overview

Asset Inventory API provides time-based storage for network assets. Once stored, assets
should be query-able by dimensions such as time, IP address, and host name. The intent is
to provide point-in-time lookup and attribution for network assets. An example use case
for this would include hydrating AWS VPC Flow Logs with identifying information.

<a id="markdown-quick-start" name="quick-start"></a>
## Quick Start

<Hello world style example.>

<a id="markdown-configuration" name="configuration"></a>
## Configuration

<Details of how to actually work with the project>

<a id="markdown-status" name="status"></a>
## Status

This project is in incubation which means we are not yet operating this tool in production
and the interfaces are subject to change.

<a id="markdown-contributing" name="contributing"></a>
## Contributing

<a id="markdown-building-and-testing" name="building-and-testing"></a>
### Building And Testing

We publish a docker image called [SDCLI](https://github.com/asecurityteam/sdcli) that
bundles all of our build dependencies. It is used by the included Makefile to help make
building and testing a bit easier. The following actions are available through the Makefile:

-   make dep

    Install the project dependencies into a vendor directory

-   make lint

    Run our static analysis suite

-   make test

    Run unit tests and generate a coverage artifact

-   make integration

    Run integration tests and generate a coverage artifact

-   make coverage

    Report the combined coverage for unit and integration tests

-   make build

    Generate a local build of the project (if applicable)

-   make run

    Run a local instance of the project (if applicable)

-   make doc

    Generate the project code documentation and make it viewable
    locally.

<a id="markdown-quality-gates" name="quality-gates"></a>
### Quality Gates

Our build process will run the following checks before going green:

-   make lint
-   make test
-   make integration
-   make coverage (combined result must be 85% or above for the project)

Running these locally, will give early indicators of pass/fail.

<a id="markdown-license" name="license"></a>
### License

This project is licensed under Apache 2.0. See LICENSE.txt for details.

<a id="markdown-contributing-agreement" name="contributing-agreement"></a>
### Contributing Agreement

Atlassian requires signing a contributor's agreement before we can accept a
patch. If you are an individual you can fill out the
[individual CLA](https://na2.docusign.net/Member/PowerFormSigning.aspx?PowerFormId=3f94fbdc-2fbe-46ac-b14c-5d152700ae5d).
If you are contributing on behalf of your company then please fill out the
[corporate CLA](https://na2.docusign.net/Member/PowerFormSigning.aspx?PowerFormId=e1c17c66-ca4d-4aab-a953-2c231af4a20b).

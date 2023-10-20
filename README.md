# GoZix Glue

[documentation-img]: https://img.shields.io/badge/godoc-reference-blue.svg?color=24B898&style=for-the-badge&logo=go&logoColor=ffffff
[documentation-url]: https://pkg.go.dev/github.com/to6ka/glue/v3
[license-img]: https://img.shields.io/github/license/gozix/glue.svg?style=for-the-badge
[license-url]: https://github.com/to6ka/glue/blob/master/LICENSE
[release-img]: https://img.shields.io/github/tag/gozix/glue.svg?label=release&color=24B898&logo=github&style=for-the-badge
[release-url]: https://github.com/to6ka/glue/releases/latest
[build-status-img]: https://img.shields.io/github/actions/workflow/status/gozix/glue/go.yml?logo=github&style=for-the-badge
[build-status-url]: https://github.com/to6ka/glue/actions
[go-report-img]: https://img.shields.io/badge/go%20report-A%2B-green?style=for-the-badge
[go-report-url]: https://goreportcard.com/report/github.com/to6ka/glue
[code-coverage-img]: https://img.shields.io/codecov/c/github/gozix/glue.svg?style=for-the-badge&logo=codecov
[code-coverage-url]: https://codecov.io/gh/gozix/glue

[![License][license-img]][license-url]
[![Documentation][documentation-img]][documentation-url]

[![Release][release-img]][release-url]
[![Build Status][build-status-img]][build-status-url]
[![Go Report Card][go-report-img]][go-report-url]
[![Code Coverage][code-coverage-img]][code-coverage-url]

The package represents a very simple and easy implementation of the extensible application on golang. The core 
components of an application are bundles that are glued together using a dependency injection container.

## Installation

```shell
go get github.com/to6ka/glue/v3
```

## Documentation

You can find documentation on [pkg.go.dev][documentation-url] and read source code if needed.
   
## Built-in DI options

| Name                  | Description                               | 
|-----------------------|-------------------------------------------|
| AsCliCommand          | Add a cli command                         |
| AsPersistentFlags     | Add custom flags to root command          |
| AsPersistentPreRunner | Add persistent pre runner to root command |

## Questions

If you have any questions, feel free to create an issue.

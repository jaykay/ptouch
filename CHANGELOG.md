# Changelog

## [1.3.0](https://github.com/jaykay/ptouch/compare/v1.2.1...v1.3.0) (2026-03-05)


### Features

* **cli:** add auto-update check and `ptouch update` command ([ba14ff2](https://github.com/jaykay/ptouch/commit/ba14ff280dced335581465d39194059de0922b88))
* **cli:** add printer config via interactive discover selector ([1bbfa5c](https://github.com/jaykay/ptouch/commit/1bbfa5c27c70ce47834477c869550604f13de442))


### Bug Fixes

* **cli:** hide unknown commit/date in version output ([4bf7b04](https://github.com/jaykay/ptouch/commit/4bf7b040860de6fe3c9841f5561f10a5c8e59dae))

## [1.2.1](https://github.com/jaykay/ptouch/compare/v1.2.0...v1.2.1) (2026-03-05)


### Bug Fixes

* **cli:** populate version from Go build info for go install ([a2355cb](https://github.com/jaykay/ptouch/commit/a2355cb77464bb61d44ae47827b20f719d654760))

## [1.2.0](https://github.com/jaykay/ptouch/compare/v1.1.0...v1.2.0) (2026-03-05)


### Features

* **cli:** add version command with build-time injection ([da992a1](https://github.com/jaykay/ptouch/commit/da992a19403cf0ba629d5e4d5b18316e7b56e89c))

## [1.1.0](https://github.com/jaykay/ptouch/compare/v1.0.0...v1.1.0) (2026-03-05)


### Features

* **docs:** switch to just-the-docs theme with sidebar navigation ([022cc3d](https://github.com/jaykay/ptouch/commit/022cc3de9791b6e3811aed3e264ef185fe7a3306))


### Bug Fixes

* **ci:** add front matter to LICENSE so Jekyll renders it as HTML ([398c195](https://github.com/jaykay/ptouch/commit/398c195ff2f7cd3a1515ebb0f8dda73c501350e0))
* **ci:** add Jekyll build step to pages workflow ([436ca0f](https://github.com/jaykay/ptouch/commit/436ca0f30a9545b62c7cd9afc554fe83adb6039f))
* **ci:** include LICENSE in pages build ([73fd24a](https://github.com/jaykay/ptouch/commit/73fd24ae4efd28800d798c18e95dc6c9423cc071))
* **ci:** preserve docs/ path structure in pages build ([6c2a04a](https://github.com/jaykay/ptouch/commit/6c2a04a99975cbc17a4fa183f77135dea650e114))
* **ci:** render LICENSE as HTML on pages site ([0d232d2](https://github.com/jaykay/ptouch/commit/0d232d2b20dcaf9f85f86f2da74f6dff3f3bacd5))
* **ci:** rewrite .md links to .html for pages, drop pretty permalinks ([e321348](https://github.com/jaykay/ptouch/commit/e32134885d1d4ba99ff9a6be80f7b69011e16b25))
* **docs:** move license link to top nav, simplify build ([de228be](https://github.com/jaykay/ptouch/commit/de228be3c6de5b2016951444e65cc75d7eaf85cd))
* **print:** account for feed margin in --width calculation ([56916b4](https://github.com/jaykay/ptouch/commit/56916b4946473ca86f2e3ae1e16442f4b8da46fc))

## 1.0.0 (2026-03-04)


### Bug Fixes

* downgrade go directive to 1.24 for golangci-lint compatibility ([762711e](https://github.com/jaykay/ptouch/commit/762711e0e92df3d2924bf5c033c29b7bcb818fef))

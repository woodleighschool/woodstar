# Changelog

## [0.4.0](https://github.com/woodleighschool/woodstar/compare/0.3.3...0.4.0) (2026-07-23)


### ⚠ BREAKING CHANGES

* **munki:** model host software state

### Features

* **deps:** update dependency lucide-react (1.25.0 → 1.26.0) ([#17](https://github.com/woodleighschool/woodstar/issues/17)) ([9e79ca9](https://github.com/woodleighschool/woodstar/commit/9e79ca956c34421867e586954fc1dad487978304))
* **deps:** update pnpm (11.15.1 → 11.16.0) ([#53](https://github.com/woodleighschool/woodstar/issues/53)) ([9a2ed6a](https://github.com/woodleighschool/woodstar/commit/9a2ed6a95a5984cd364536cdcbc2db455d5aa791))
* **munki:** model host software state ([8f1ab94](https://github.com/woodleighschool/woodstar/commit/8f1ab940fbf16ab9de8d5e1ca6b8cd3fabe353f5))
* **munki:** simplify distribution point setup ([8c1bb2f](https://github.com/woodleighschool/woodstar/commit/8c1bb2f708efaed3e9f826b5bfa63670fb40b204))


### Bug Fixes

* **deps:** update dependency docusaurus-plugin-llms (0.5.0 → 0.5.1) ([#26](https://github.com/woodleighschool/woodstar/issues/26)) ([15975fb](https://github.com/woodleighschool/woodstar/commit/15975fbecfe6a382778975d84bdc9cb91b02cd60))
* **deps:** update dependency sass (1.101.3 → 1.101.4) ([#55](https://github.com/woodleighschool/woodstar/issues/55)) ([7eb07df](https://github.com/woodleighschool/woodstar/commit/7eb07df117f8dcd41b95678be95268a871d81958))
* **deps:** update dependency sass (1.101.4 → 1.101.6) ([#56](https://github.com/woodleighschool/woodstar/issues/56)) ([9aaad3f](https://github.com/woodleighschool/woodstar/commit/9aaad3f1f678565688bd1175aa940d4334432390))
* **deps:** update module buf.build/gen/go/northpolesec/protos/protocolbuffers/go (v1.36.11-20260721180550-e61e39a61420.1 → v1.36.11-20260723221051-096a321dccc8.1) ([#60](https://github.com/woodleighschool/woodstar/issues/60)) ([94d712e](https://github.com/woodleighschool/woodstar/commit/94d712ebe0b87e0d40138c923838a32b05031f3e))
* **deps:** update module github.com/gabriel-vasile/mimetype (v1.4.14 → v1.4.15) ([#59](https://github.com/woodleighschool/woodstar/issues/59)) ([0feec09](https://github.com/woodleighschool/woodstar/commit/0feec091b39d356d27a37c605086afea21e3d808))
* **deps:** update module github.com/pressly/goose/v3 (v3.27.2 → v3.27.3) ([#45](https://github.com/woodleighschool/woodstar/issues/45)) ([5de1b5e](https://github.com/woodleighschool/woodstar/commit/5de1b5eec2e9bab58c7b48cacbb9d97906301854))
* **web:** bound nested data tables ([b0dd100](https://github.com/woodleighschool/woodstar/commit/b0dd1009b10ceff9658de5292fa73b0107a2ad7a))


### Documentation

* **munki:** explain client IP matching ([b99a4b8](https://github.com/woodleighschool/woodstar/commit/b99a4b8e01fef707acb04a11bc73292b068b2c18))


### Code Refactoring

* **storage:** use one S3 endpoint ([35b4f42](https://github.com/woodleighschool/woodstar/commit/35b4f42207ee27d4a6de85ffa288ea5cac292122))
* **web:** use generic route IDs ([4bd17f9](https://github.com/woodleighschool/woodstar/commit/4bd17f9ba07e69d535a0c0431e8166c31d02b4f4))

## [0.3.3](https://github.com/woodleighschool/woodstar/compare/0.3.2...0.3.3) (2026-07-23)


### Features

* **release:** ship binaries and MDP worker ([3eaacdb](https://github.com/woodleighschool/woodstar/commit/3eaacdb782522daa09c98083e027c1cd3638dd05))


### Code Refactoring

* **web:** make page titles action-oriented ([177c715](https://github.com/woodleighschool/woodstar/commit/177c7154d60062fae8f01f2f076de6937b42252b))

## [0.3.2](https://github.com/woodleighschool/woodstar/compare/0.3.1...0.3.2) (2026-07-23)


### Bug Fixes

* **autopkg:** upload packages with curl ([ee2b2dd](https://github.com/woodleighschool/woodstar/commit/ee2b2dd91a97c52be22708377670e6e3e36eec5b))
* **web:** compose Santa event tabs as links ([c680ae3](https://github.com/woodleighschool/woodstar/commit/c680ae3738624f7e0dd4f18dc684d83f56c78ae3))
* **web:** correct checkbox disabled state ([b5fa648](https://github.com/woodleighschool/woodstar/commit/b5fa648d760e7a862e4a06cc75ad5fd4d855598b))
* **web:** keep table pagination from submitting forms ([9150852](https://github.com/woodleighschool/woodstar/commit/915085263f166d248b1725d22fb49bf5133a4ceb))


### Code Refactoring

* **labels:** use shared data table ([317de9c](https://github.com/woodleighschool/woodstar/commit/317de9c2ca78c99f958910c9c3f8eef30a1daef2))
* **web:** use the native sidebar shell ([b78a65c](https://github.com/woodleighschool/woodstar/commit/b78a65cef2f2f749d36f440c9194933ade0db432))

## [0.3.1](https://github.com/woodleighschool/woodstar/compare/0.3.0...0.3.1) (2026-07-23)


### Features

* **deps:** update dependency shadcn (4.13.1 → 4.14.0) ([#49](https://github.com/woodleighschool/woodstar/issues/49)) ([056d5e7](https://github.com/woodleighschool/woodstar/commit/056d5e716ab7a7e7b7ec0291d58143d8599fe2f6))


### Bug Fixes

* **deps:** update dependency @vitejs/plugin-react (6.0.3 → 6.0.4) ([#46](https://github.com/woodleighschool/woodstar/issues/46)) ([155d6d7](https://github.com/woodleighschool/woodstar/commit/155d6d72de41527ce4b04f3a09ab19fa0653e50b))
* **deps:** update module github.com/gabriel-vasile/mimetype (v1.4.13 → v1.4.14) ([#44](https://github.com/woodleighschool/woodstar/issues/44)) ([3ff07cc](https://github.com/woodleighschool/woodstar/commit/3ff07ccb62031a83eafbcb539783d3e8ffd8a55b))
* **web:** constrain client resources preview ([0a90b4d](https://github.com/woodleighschool/woodstar/commit/0a90b4d3fe9da2be040881362b05531610060afc))

## [0.3.0](https://github.com/woodleighschool/woodstar/compare/0.2.0...0.3.0) (2026-07-22)


### ⚠ BREAKING CHANGES

* **deps:** Update dependency oxlint-tsgolint (0.25.0 → 7.0.2001) ([#39](https://github.com/woodleighschool/woodstar/issues/39))

### Features

* **deps:** update aws-sdk-go-v2 monorepo ([#38](https://github.com/woodleighschool/woodstar/issues/38)) ([0632fd7](https://github.com/woodleighschool/woodstar/commit/0632fd7f8c5ce2e88f6cfbba9cb6c0bb565fd490))
* **deps:** Update dependency oxlint-tsgolint (0.25.0 → 7.0.2001) ([#39](https://github.com/woodleighschool/woodstar/issues/39)) ([3babbc5](https://github.com/woodleighschool/woodstar/commit/3babbc513570f22c7e693108dba5298e98001453))
* **munki:** expose client resources as CRUD ([20233bf](https://github.com/woodleighschool/woodstar/commit/20233bfa9c228624f5effbd2ef363c92cb2887d2))


### Bug Fixes

* **autopkg:** allow nopkg and fix ssl cert input ([f7bc94a](https://github.com/woodleighschool/woodstar/commit/f7bc94a7521671c3ec2e57ba6d2ea2832cf45495))
* **deps:** update dependency react (19.2.7 → 19.2.8) ([#41](https://github.com/woodleighschool/woodstar/issues/41)) ([056ccc6](https://github.com/woodleighschool/woodstar/commit/056ccc6eaefdfe91cf9ccc9844f2758cd2415902))
* **deps:** update dependency react-dom (19.2.7 → 19.2.8) ([#42](https://github.com/woodleighschool/woodstar/issues/42)) ([1328fe6](https://github.com/woodleighschool/woodstar/commit/1328fe644cfdbe5b832b1c5b5b74a572f7b97268))
* **storage:** upload directly to object keys ([e9e8758](https://github.com/woodleighschool/woodstar/commit/e9e875829b93f176252aa645b6d1761761605705))
* **web:** restore larger page size options ([de8d9d4](https://github.com/woodleighschool/woodstar/commit/de8d9d4219b72f84423cf17b97e67a1cc6b8764e))
* **web:** truncate long table paths ([df36036](https://github.com/woodleighschool/woodstar/commit/df3603634147898741d71a24d4389b5993e205bd))


### Documentation

* deploy site to GitHub Pages ([2d4e1f9](https://github.com/woodleighschool/woodstar/commit/2d4e1f98d73243eb504e82ca679ddaf36255d45f))
* link to published site ([34f2e58](https://github.com/woodleighschool/woodstar/commit/34f2e589f582fe6b1f10e2476f310d9dc878951b))
* remove footer copyright ([90e2fef](https://github.com/woodleighschool/woodstar/commit/90e2fef418499fb5fd6b46779d43566f92bf5fb1))


### Code Refactoring

* **handlers:** remove missing message parameters from host state registration ([451aadf](https://github.com/woodleighschool/woodstar/commit/451aadf61617deec6ec21836e2e9e13184db333b))

## [0.2.0](https://github.com/woodleighschool/woodstar/compare/v0.1.8...0.2.0) (2026-07-22)


### ⚠ BREAKING CHANGES

* **deps:** Update Node.js (v20.20.2 → v24.18.0) ([#29](https://github.com/woodleighschool/woodstar/issues/29))

### Features

* **auth:** manage users through the CLI ([ab0e433](https://github.com/woodleighschool/woodstar/commit/ab0e4331c3e0856a2b9e78a9f0b9e7816bcc328b))
* **auth:** replace setup with configured administrator ([d34bdb1](https://github.com/woodleighschool/woodstar/commit/d34bdb1565484efc2b729c25a0b23d06048403e6))
* **autopkg:** upload Munki items directly ([8e42906](https://github.com/woodleighschool/woodstar/commit/8e42906440daac89adf36facc9d0bff2b7cc0e72))
* **cli:** run server from root command ([6989b42](https://github.com/woodleighschool/woodstar/commit/6989b422bc3cb2cefdfa2ed59718b934a814c069))
* **deps:** update module github.com/testcontainers/testcontainers-go (v0.42.0 → v0.43.0) ([#10](https://github.com/woodleighschool/woodstar/issues/10)) ([5646226](https://github.com/woodleighschool/woodstar/commit/5646226c2a152d0d5538e0abcde16a6d33bbd028))
* **deps:** update node.js (v20.0.0 → v20.20.2) ([#11](https://github.com/woodleighschool/woodstar/issues/11)) ([1cb8280](https://github.com/woodleighschool/woodstar/commit/1cb8280ba2967f38f94f956ce36a0c6381983adf))
* **deps:** Update Node.js (v20.20.2 → v24.18.0) ([#29](https://github.com/woodleighschool/woodstar/issues/29)) ([f3628f1](https://github.com/woodleighschool/woodstar/commit/f3628f1b9d24ae5a454860dfedb112911af2b789))
* **munki:** nest software in package models ([81d77be](https://github.com/woodleighschool/woodstar/commit/81d77be99d93b66d0452d9398f0792b40ded5920))
* **munki:** support custom client resources ([8be3acc](https://github.com/woodleighschool/woodstar/commit/8be3acce1ce179bb4c7a45c4b6f7e40594087283))
* **storage:** own object transfer lifecycle ([3720b78](https://github.com/woodleighschool/woodstar/commit/3720b787101b391ab8ef56bb3a89b3f54fc7678f))
* **web:** unify software icon rendering ([77f904e](https://github.com/woodleighschool/woodstar/commit/77f904e98afbfb39ccda6ebcf6e56caa211ea801))


### Bug Fixes

* **api:** enforce generated client contracts ([6c67ef5](https://github.com/woodleighschool/woodstar/commit/6c67ef5b2e46a77172cf0e8afc14ddf84ee791b5))
* **auth:** rate limit password login ([2e10b19](https://github.com/woodleighschool/woodstar/commit/2e10b19455986adc11fc1b3d6dd5d140fac8f1fc))
* **autopkg:** batch package cleanup ([bd28822](https://github.com/woodleighschool/woodstar/commit/bd28822ac82cc16379329a489e88543bb3c4d19e))
* **autopkg:** use icon upload endpoint ([57ddebe](https://github.com/woodleighschool/woodstar/commit/57ddebe0edb4f28065ffd34780548caffc8ae9da))
* **container:** update image golang (1.26.4 → 1.26.5) ([#1](https://github.com/woodleighschool/woodstar/issues/1)) ([31d7a3b](https://github.com/woodleighschool/woodstar/commit/31d7a3b849bcdeebb0f1a007a49f0dd5fbfa09f7))
* **dbutil:** normalize JSON slice encoding ([c9a7418](https://github.com/woodleighschool/woodstar/commit/c9a74183dfc193fc0b42638726f6a876ba8d1f15))
* **deps:** update dependency sass (1.101.0 → 1.101.3) ([#37](https://github.com/woodleighschool/woodstar/issues/37)) ([0d11c26](https://github.com/woodleighschool/woodstar/commit/0d11c26f354c547ae1de271cf85e4509099aa240))
* **deps:** update dependency shadcn (4.13.0 → 4.13.1) ([#24](https://github.com/woodleighschool/woodstar/issues/24)) ([b9a0e9e](https://github.com/woodleighschool/woodstar/commit/b9a0e9e536bb33c3de97aeb50826912c7857e9d2))
* **deps:** update dependency vite (8.1.4 → 8.1.5) ([#2](https://github.com/woodleighschool/woodstar/issues/2)) ([f40a3b5](https://github.com/woodleighschool/woodstar/commit/f40a3b59993f8d3483ca19d2489b8d7daf600a82))
* **deps:** update module buf.build/gen/go/northpolesec/protos/protocolbuffers/go (v1.36.11-20260715141808-2e443f37a544.1 → v1.36.11-20260716230937-b90c682e2e73.1) ([#3](https://github.com/woodleighschool/woodstar/issues/3)) ([f77c7fc](https://github.com/woodleighschool/woodstar/commit/f77c7fc5b9c90cf43c1ed41539342606e68e0b28))
* **deps:** update module buf.build/gen/go/northpolesec/protos/protocolbuffers/go (v1.36.11-20260716230937-b90c682e2e73.1 → v1.36.11-20260721180550-e61e39a61420.1) ([#34](https://github.com/woodleighschool/woodstar/issues/34)) ([bdc8f87](https://github.com/woodleighschool/woodstar/commit/bdc8f87c056708d76d7767490f39ee60477dfd50))
* **deps:** update module github.com/aws/aws-sdk-go-v2/service/s3 (v1.105.1 → v1.105.2) ([#4](https://github.com/woodleighschool/woodstar/issues/4)) ([e9b1fe6](https://github.com/woodleighschool/woodstar/commit/e9b1fe627da40c14f6083ba3d1b4ee77b1d19e46))
* **deps:** update module github.com/aws/smithy-go (v1.27.3 → v1.27.4) ([#5](https://github.com/woodleighschool/woodstar/issues/5)) ([8c68758](https://github.com/woodleighschool/woodstar/commit/8c68758e813bffe27135e500a44dfd5304ae2163))
* **deps:** update pnpm (11.13.0 → 11.13.1) ([#6](https://github.com/woodleighschool/woodstar/issues/6)) ([abaf46e](https://github.com/woodleighschool/woodstar/commit/abaf46e57edd9a015101c72864054f5a74a41bca))
* **deps:** update pnpm (11.15.0 → 11.15.1) ([#27](https://github.com/woodleighschool/woodstar/issues/27)) ([3e5acf3](https://github.com/woodleighschool/woodstar/commit/3e5acf370bdcc988b52f0aeaaec06ed2953429ed))
* **deps:** update tailwindcss monorepo (4.3.2 → 4.3.3) ([#7](https://github.com/woodleighschool/woodstar/issues/7)) ([a892001](https://github.com/woodleighschool/woodstar/commit/a89200106fee738cf993d24dbb2d3644d25134de))
* **deps:** update tanstack-query monorepo (5.101.2 → 5.101.4) ([#36](https://github.com/woodleighschool/woodstar/issues/36)) ([6caf0dc](https://github.com/woodleighschool/woodstar/commit/6caf0dc5bcabbda34ad868af0cde2fb04cdd1c64))
* **directory:** preserve identity casing ([61eef26](https://github.com/woodleighschool/woodstar/commit/61eef265036cdb07c1bbbe0b5afd34ef1862ac46))
* **munki:** align software and package forms ([0f9e9e8](https://github.com/woodleighschool/woodstar/commit/0f9e9e89a14f002fed339ea7822f35b483e09925))
* **munki:** make upload strategy contracts total ([441f102](https://github.com/woodleighschool/woodstar/commit/441f102e9a79bc8179f7bcd46711d7a969d294de))
* **munki:** refresh software-backed projections ([5d9bd7c](https://github.com/woodleighschool/woodstar/commit/5d9bd7c7b38ed831db4fc7b790afe5283a6ed7ba))
* **persistence:** own cleanup and label refresh lifecycles ([1388e08](https://github.com/woodleighschool/woodstar/commit/1388e0874ce8c6c499663618aee81ec4f19981a3))
* **santa:** preserve unsigned sync values ([574eb7d](https://github.com/woodleighschool/woodstar/commit/574eb7d950d4c112a09c1bb9e5b4508d0ea5b8df))
* **security:** enforce browser response policy ([f99c9d9](https://github.com/woodleighschool/woodstar/commit/f99c9d9f9eba2743650ba962cced41f70e62b160))
* **storage:** own transfer capability keys ([38b0e59](https://github.com/woodleighschool/woodstar/commit/38b0e599452b234e2754670609063fd367c8b7af))
* **storage:** simplify object cleanup ([7bef771](https://github.com/woodleighschool/woodstar/commit/7bef771e91a1afcab5538b1400920869e8e926f4))
* **web:** align API and lint contracts ([59b724e](https://github.com/woodleighschool/woodstar/commit/59b724e5a89da53a7d63dec41a00e173914e46a1))
* **web:** correct table controls and package fields ([fc7bdda](https://github.com/woodleighschool/woodstar/commit/fc7bddae85def62ef7ac1937da3b6651ada8d901))
* **web:** derive editor contracts from Zod ([9c7aba1](https://github.com/woodleighschool/woodstar/commit/9c7aba197e775569eacfef5600ee9c27cfab9192))
* **web:** keep dynamic breadcrumbs live ([b18bddf](https://github.com/woodleighschool/woodstar/commit/b18bddf0520329ed3cd102fbf9b759cf5802c583))
* **web:** normalize form exit behavior ([3106170](https://github.com/woodleighschool/woodstar/commit/3106170f4855454052650c3bfbde4c5f4905d0e2))
* **web:** remove discard prompts from dialogs ([f2324cf](https://github.com/woodleighschool/woodstar/commit/f2324cf779c62b5cf0596a1df71eb0c83d1f9cf6))
* **web:** show spinners for pending actions ([313031e](https://github.com/woodleighschool/woodstar/commit/313031e24828c5d3836d68ae2d746adca62bd6b8))
* **web:** split editor form responsibilities ([6b7cd13](https://github.com/woodleighschool/woodstar/commit/6b7cd13cadb35f0f5e2828832d9b241ba8fd358f))


### Code Refactoring

* **api:** declare route transport policy ([13fc982](https://github.com/woodleighschool/woodstar/commit/13fc982df3dcfdf38cc78d3d090cc1d25fd0e8c2))
* **api:** normalize admin resource routes ([b15f5fe](https://github.com/woodleighschool/woodstar/commit/b15f5fe61f5e6fd072a7688ff99e6128f61ae899))
* **api:** simplify server composition ([919c2a9](https://github.com/woodleighschool/woodstar/commit/919c2a9670515f8a82d2806345665314bf988c39))
* inline trivial wrappers ([b4fa6ee](https://github.com/woodleighschool/woodstar/commit/b4fa6ee4b887e4abda594bdc02f882fefe5a0834))
* split package files by concern ([249876a](https://github.com/woodleighschool/woodstar/commit/249876ab850150cc000d5b0aa3d637c80c9265f9))
* **web:** remove remote image decoration ([2b3ece8](https://github.com/woodleighschool/woodstar/commit/2b3ece8a96edfd3a85bc426fa3268b04ac545415))
* **web:** unify route and query boundaries ([0b12f1a](https://github.com/woodleighschool/woodstar/commit/0b12f1ab2da3f04b525c709aed13fb23f3d1b936))
* **web:** use system theme only ([11bdbff](https://github.com/woodleighschool/woodstar/commit/11bdbff88f3f6b3f260ad6a76af3b48eb907c57a))
* **web:** use vendored byte formatter ([5d2cf84](https://github.com/woodleighschool/woodstar/commit/5d2cf84f9c9349e2801c425cb13eae2382f868bd))

# goswim - A Shallow RESTful api for Ansible, Terraform ...
... and basically anything you would like to run as jobs in docker containers, authenticated with Hashicorp Vault AppRoles with Secret Injection.

Goal is to be a Highly Available and Scaleable API for automation.

See [Concept Ideas](docs/Concept_Ideas.md)

At this stage this project is a proof-of-concept and under development...

See [build_test_dev script](./build_test_against_dev.sh) for example starting the goswim docker container with the instances of Vault and MongoDb running in the vagrant container.

[Dev Notes](docs/devnotes.nd)


## LICENSE - GPLv3

```
Copyright 2018 Graham Lee Bevan <graham.bevan@ntlworld.com>

goswim is free software: you can redistribute it and/or modify
it under the terms of the GNU General Public License as published by
the Free Software Foundation, either version 3 of the License, or
(at your option) any later version.

goswim is distributed in the hope that it will be useful,
but WITHOUT ANY WARRANTY; without even the implied warranty of
MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE.  See the
GNU General Public License for more details.

You should have received a copy of the GNU General Public License
along with goswim.  If not, see <https://www.gnu.org/licenses/>.
```

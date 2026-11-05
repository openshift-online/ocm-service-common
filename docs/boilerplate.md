## Boilerplate

This repository subscribes to the `openshift/rosa-services` boilerplate convention in
[`boilerplate/update.cfg`](boilerplate/update.cfg). It supplies the shared root
`OWNERS_ALIASES`; this repository maintains its own `OWNERS` files, which can
reference those aliases. See the [convention README](boilerplate/openshift/rosa-services/README.md)
for details.

To pull in changes to the convention (including alias membership), run
`make boilerplate-update` from a clean checkout, review the resulting changes,
and commit them. Update shared aliases in the upstream
[openshift/boilerplate](https://github.com/openshift/boilerplate) repository rather
than editing the copied `OWNERS_ALIASES` or managed boilerplate files here.

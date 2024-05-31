# Documentation

The documentation is served:

* As a whole from [this GitHub page](https://nikolalohinski.github.io/terraform-provider-freebox)
* From the [`terraform` provider registry](https://registry.terraform.io/providers/NikolaLohinski/freebox/latest) for provider usage information.

The following can be used to generate the registry documentation in the `docs/` folder:

```shell
mage docs:build
```

And served locally within [an `mdBook`](https://rust-lang.github.io/mdBook) at [http://localhost:3000](http://localhost:3000) using:

```shell
mage book:serve
```

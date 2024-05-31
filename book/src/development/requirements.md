# Requirements

* Install `go >= 1.21` by following the [official documentation](https://go.dev/doc/install) ;

* Install `mage` using the [online documentation](https://magefile.org) ;

* Run the following to fetch all the required tools:

  ```sh
  mage install
  ```

* Define the following environment variables required for integration testing:
  
  ```sh
  export FREEBOX_ENDPOINT="http://mafreebox.freebox.fr" # 👈 Or set it to your external endpoint if not running on your home network
  export FREEBOX_VERSION="latest"
  export FREEBOX_APP_ID="..." # See https://dev.freebox.fr/sdk/os/login/ to learn
  export FREEBOX_TOKEN="..." #  how to define an app and generate a private token
  ```

* Verify the previous steps by running:

  ```sh
  mage
  ```
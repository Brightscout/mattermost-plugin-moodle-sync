# Mattermost Moodle Sync Plugin
## Installation

1. You can get the latest version on the [releases page](https://github.com/Brightscout/x-mattermost-plugin-moodle-sync/releases).
1. Upload this file in the Mattermost **System Console > Plugins > Management** page to install the plugin. To learn more about how to upload a plugin, [see the documentation](https://docs.mattermost.com/administration/plugins.html#custom-plugins).
1. After installing the plugin, you should go to the plugin's settings in System Console and set the Webhook Secret (more about this below).


### System Console Settings

- **Webhook Secret**:
  Setting a webhook secret allows you to ensure that the requests sent to the payload URL are from Moodle, and is used with every request that is made from Moodle to Mattermost.

  **Moodle Bot Username**
  Set the username for the moodle bot which will be a member of every channel made by Moodle and will notify you everytime a user's role is updated in a channel.

  **Moodle Bot Display Name**
  Set the display name for the moodle bot.

  **Moodle Bot Description**
  Set the description for the moodle bot.

## Building the plugin

- Make sure you have following components installed:
    - Go - v1.16 - [Getting Started](https://golang.org/doc/install)
      > **Note:** If you have installed Go to a custom location, make sure the `$GOROOT` variable is set properly. Refer [Installing to a custom location](https://golang.org/doc/install#install).
    - NodeJS - v14.17 and NPM - [Downloading and installing Node.js and npm](https://docs.npmjs.com/getting-started/installing-node).
    - Make

- Note that this project uses [Go modules](https://github.com/golang/go/wiki/Modules). Be sure to locate the project outside of `$GOPATH`.
To learn more about plugins, see [plugin documentation](https://developers.mattermost.com/extend/plugins/).

- Build your plugin:
    ```
    make dist
    ```

- This will produce a single plugin file (with support for multiple architectures) for upload to your Mattermost server:
    ```
    dist/com.mattermost.moodle-sync-x.y.z.tar.gz
    ```

## Development

To avoid having to manually install your plugin, build and deploy your plugin using one of the following options.

### Deploying with Local Mode

If your Mattermost server is running locally, you can enable [local mode](https://docs.mattermost.com/administration/mmctl-cli-tool.html#local-mode) to streamline deploying your plugin. Edit your server configuration as follows:

```json
{
    "ServiceSettings": {
        ...
        "EnableLocalMode": true,
        "LocalModeSocketLocation": "/var/tmp/mattermost_local.socket"
    }
}
```

and then deploy your plugin:
```
make deploy
```

You may also customize the Unix socket path:
```
export MM_LOCALSOCKETPATH=/var/tmp/alternate_local.socket
make deploy
```

### Deploying with credentials

Alternatively, you can authenticate with the server's API with credentials:
```
export MM_SERVICESETTINGS_SITEURL=http://localhost:8065
export MM_ADMIN_USERNAME=admin
export MM_ADMIN_PASSWORD=password
make deploy
```

or with a [personal access token](https://docs.mattermost.com/developer/personal-access-tokens.html):
```
export MM_SERVICESETTINGS_SITEURL=http://localhost:8065
export MM_ADMIN_TOKEN=j44acwd8obn78cdcx7koid4jkr
make deploy
```

---

Made with &#9829; by [Brightscout](http://www.brightscout.com)

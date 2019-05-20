# mattermost-plugin-giphy
This Mattermost plugin adds Giphy Integration by creating a `/gif` slash command (no webhooks or additional installation required).

## COMPATIBILITY
- for Mattermost 5.10 or higher: use v1.x.x release (needs to be abe to put button on ephemeral posts)
- for Mattermost 5.2 to 5.9: use v0.2.0 release
- for Mattermost 4.6 to 5.1: use v0.1.x release
- for Mattermost below: unsupported versions (plugins can't create slash commands)

## Installation and configuration
1. Go to the [Releases page](https://github.com/moussetc/mattermost-plugin-giphy/releases) and download the package for your OS and architecture.
2. Use the Mattermost `System Console > Plugins Management > Management` page to upload the `.tar.gz` package
3. Go to the `System Console > Plugins > Giphy` configuration page that appeared, and configure the Giphy API key. The default key is the [public beta key, which is 'subject to rate limit constraints'](https://developers.giphy.com/docs/).
4. You can also configure the following settings :
    - rating
    - language (see [Giphy Language support](https://developers.giphy.com/docs/#rendition-guide))
    - display size (see [Giphy rendition guide](https://developers.giphy.com/docs/#rendition-guide))
4. **Activate the plugin** in the `System Console > Plugins Management > Management` page

## Manual configuration
If you need to enable & configure this plugin directly in the Mattermost configuration file `config.json`, for example if you are doing a [High Availability setup](https://docs.mattermost.com/deployment/cluster.html), you can use the following lines (remember to set the API key!):
```json
 "PluginSettings": {
        // [...]
        "Plugins": {
            "com.github.moussetc.mattermost.plugin.giphy": {
                "apikey": "YOUR_GIPHY_API_KEY_HERE", 
                "language": "en",
                "rating": "",
                "rendition": "fixed_height_small"
            },
        },
        "PluginStates": {
            // [...]
            "com.github.moussetc.mattermost.plugin.giphy": {
                "Enable": true
            },
        }
    }
```

## Usage
- The `/gif cute doggo` command will make a post with a GIF from Giphy that matches the 'cute doggo' query. The GIF will be directly posted, using the user's avatar and name.
- The '/gifs happy kitty' command will post a private message with a GIF. You can choose to post this GIF, or shuffle to get another one that matches the query.

## TROUBLESHOOTING
- Is your plugin version compatible with your server version? Check the Compatibility section in the README.
- Make sure you have configured the SiteURL setting correctly in the Mattermost administration panel.
- If you get the following error : `{"level":"error", ... ,"msg":"Unable to get GIF URL", ... ,"method":"POST","err_where":"Giphy Plugin","http_code":400,"err_details":"Error HTTP status 429: 429 Unknown Error"}`: the `429` HTTP status code indicate that you have exceeded the allowed requests for your API key. *Make sure your API is valid for your usage.* This typically happens with the default API key, which musn't be used in production.

## Development
Run make vendor to install dependencies, then develop like any other Go project, using go test, go build, etc.

If you want to create a fully bundled plugin that will run on a local server, you can use make `mattermost-plugin-giphy.tar.gz`.

## What's next?
- Allow customization of the trigger command
- Command to choose between several GIFS at once (alternative to shuffle)

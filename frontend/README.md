# listmonk frontend (Vue + Buefy)

It's best if the `listmonk/frontend` directory is opened in an IDE as a separate project where the frontend directory is the root of the project.

For developer setup instructions, refer to the main project's README.

## Globals
In `main.js`, Buefy and vue-i18n are attached globally. In addition:

- `$api` (collection of API calls from `api/index.js`)
- `$utils` (util functions from `util.js`). They are accessible within Vue as `this.$api` and `this.$utils`.

Some constants are defined in `constants.js`.


## APIs and states
The project uses a global `vuex` state to centrally store the responses to pretty much all APIs (eg: fetch lists, campaigns etc.) except for a few exceptions. These are called `models` and have been defined in `constants.js`. The definitions are in `store/index.js`.

There is a global state `loading` (eg: loading.campaigns, loading.lists) that indicates whether an API call for that particular "model" is running. This can be used anywhere in the project to show loading spinners for instance. All the API definitions are in `api/index.js`. It also describes how each API call sets the global `loading` status alongside storing the API responses.

*IMPORTANT*: All JSON field names in GET API responses are automatically camel-cased when they're pulled for the sake of consistency in the frontend code and for complying with the linter spec in the project (Vue/AirBnB schema). For example, `content_type` becomes `contentType`. When sending responses to the backend, however, they should be snake-cased manually. This is overridden for certain calls such as `/api/config` and `/api/settings` using the `preserveCase: true` param in `api/index.js`.


## Icon pack
Buefy by default uses [Material Design Icons](https://materialdesignicons.com) (MDI) with icon classes prefixed by `mdi-`.

listmonk uses only a handful of icons from the massive MDI set packed as web font, using [Fontello](https://fontello.com). To add more icons to the set using fontello:

- Go to Fontello and drag and drop `frontend/fontello/config.json` (This is the full MDI set converted from TTF to SVG icons to work with Fontello).
- Use the UI to search for icons and add them to the selection (add icons from under the `Custom` section)
- Download the Fontello pack and from the ZIP:
    - Copy and overwrite `config.json` to `frontend/fontello`
    - Copy `fontello.woff2` to `frontend/src/assets/icons`.
    - Open `css/fontello.css` and copy the individual icon definitions and overwrite the ones in `frontend/src/assets/icons/fontello.css`

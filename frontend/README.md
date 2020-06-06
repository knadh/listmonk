# listmonk frontend (Vue + Buefy)

It's best if the `listmonk/frontend` editor is opened in an IDE as a separate project where the frontend directory is the rool of the project.


## Icon pack
Buefy by default uses [Material Design Icons](https://materialdesignicons.com) (MDI) with icon classes prefixed by `mdi-`.

listmonk uses only a handful of icons from the massive MDI set packed as web font, using [Fontello](https://fontello.com). To add more icons to the set using fontello:

- Go to Fontello and drag and drop `frontend/fontello/config.json` (This is the full MDI set converted from TTF to SVG icons to work with Fontello).
- Use the UI to search for icons and add them to the selection (add icons from under the `Custom` section)
- Download the Fontello pack and from the ZIP:
    - Copy and overwrite `config.json` to `frontend/fontello`
    - Copy `fontello.woff2` to `frontend/src/assets/icons`.
    - Open `css/fontello.css` and copy the individual icon definitions and overwrite the ones in `frontend/src/assets/icons/fontello.css`

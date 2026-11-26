# shellfish

A virtual pet Fish that lives in your terminal. You name it, feed it and keep it company, and it carries on living while the app is closed: it gets hungry in real time, but it never dies.

<!-- A screenshot goes here: docs/screenshot.png, taken in a real terminal. -->

## Install

**Homebrew** (macOS and Linux):

```sh
brew install kyu-seok/tap/shellfish
```

**npm:**

```sh
npm install -g @kyu-seok/shellfish
# or try it without installing:
npx @kyu-seok/shellfish
```

**Go** (1.24 or later):

```sh
go install github.com/Kyu-seok/shellfish@latest
```

Or download a binary for macOS, Linux or Windows from the [Releases page](https://github.com/Kyu-seok/shellfish/releases).

## Play

Run `shellfish`. The first time, pick a Species and give your Fish a name.

| Do this | And |
| --- | --- |
| Click the water | Drop Food where you clicked |
| `f` or space | Drop Food at the top of the Tank |
| `1`, `2`, `3` | Choose the Food: pellet, flake or worm |
| Click the Fish | Pat it: it wiggles and sends up a heart |
| Move the pointer over the Tank | A curious Fish swims over to see |
| `r` | Release the Fish, to start again with a new one |
| `q` or Esc | Close the Tank (the Fish carries on living) |

The Fish shows how it feels: Content, Hungry or Starving. A hungry Fish dashes for Food, and a Starving one droops to the bottom of the Tank until it's fed.

The Tank needs a window of at least 30×10 and a terminal with 256 colours; with fewer, or with `NO_COLOR` set, the Fish is drawn as a silhouette. In tmux, turn on `set -g mouse on` for clicking and the pointer.

`shellfish --version` and `shellfish --help` work any time, even while a Tank is open.

## Where your Fish lives

One Fish is saved per user, in `shellfish/fish.json` under:

- macOS: `~/Library/Application Support`
- Linux: `$XDG_DATA_HOME`, or `~/.local/share`
- Windows: `%LOCALAPPDATA%`

If `$XDG_DATA_HOME` is set, it is used on any system.
Only one Tank can be open at a time.

## License

[MIT](LICENSE)

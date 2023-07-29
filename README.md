# Plex-Richpresence

> Note: This was heavily inspired from [Ombrelin's Plex Rich Presence](https://github.com/Ombrelin/plex-rich-presence) (a colleague in the same school as me 😀) in Java. Since I disliked Java and the idea that I needed to install the right version of something, + because the multi-servers support was somewhat flaky, I decided to create mine in Golang.

# Features

General:
- Start on boot
- Transparent auto-updater
- Multilanguages (currently only English and French, feel free to add a new one!)

Plex:
- SSO integration
- Work for unexposed servers (but they still need to be owned by a Plex account)
- Select which player you want to track (a player is added after the content is played for the first time on it)
- Track new players automatically or not
- Track all the servers that you own/are invited to

Rich Presence:
- Cover art/Thumbnail display
- Elapsed or remaining time display
- Buttons for music details (not 100% stable, some links are broken)

# Downloads

Everything is available in the releases here: https://github.com/Arno500/plex-richpresence/releases

# Using

Download the latest binary [here](https://github.com/Arno500/plex-richpresence/releases), store it where you want and run it. That's it.  
If needed, the config file should be in your user folder, in an `Arno & Co` subfolder (`AppData/Roaming` in Windows, `.config` in Linux, etc...). It only stores Plex auth informations, to allow quickly running the app and get it working out-of-the-box.
Logs are also here, so if you need to open an issue, don't forget to include them! (and erase any potential sensitive informations like your Plex hostnames, etc...)  
You can also set some options in the tray icon, available in the notification area in Windows, or the tray area in macOS and various Linux DE.

# Building

You'll need GCC (even on Windows, unfortunately)

## Windows

`go build -tags windows -ldflags='-s -w -H=windowsgui'`

## Linux

`go build -tags unix -ldflags='-s -w'`

## MacOS

`go build -tags darwin -ldflags='-s -w'`

# Contributing

## Locales

I use https://github.com/nicksnyder/go-i18n. Please follow their instructions to add a language file first.

1. `goi18n extract --format json -outdir i18n/locales` to extract any string you added in the program
2. `goi18n merge --format json -outdir i18n/locales i18n/locales/active.en.json i18n/locales/active.fr.json` to create the delta between english and french
3. Translate everything in `i18n/locales/translate.fr.json`
4. `goi18n merge --format json -outdir i18n/locales i18n/locales/active.en.json i18n/locales/active.fr.json i18n/locales/translate.fr.json` Re-run the command to include the new strings in the active file
5. Remove the `translate.xx.json` file

## Packaging

https://github.com/markbates/pkger is used to embed files in the executable itself, notably locale files. The package will detect the needed files when compiling the executable, so nothing to do here.

## Windows manifest

`rsrc -ico Plex_IDI_ICON1.ico -manifest plex-richpresence.manifest`

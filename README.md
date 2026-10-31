# Kiko Sfos

> SailfishOS Driver client for Kiko. Yes.

## What's inside

Inside of this magnificent work you can find

1) App for SailfishOS
2) Native integration with
  a) Geoclue - for GPS positioning
  b) PureMaps - for navigation
  c) ADA - through blinklabs.io's open source solutions
  d) BSV - through official `https://github.com/bsv-blockchain/go-sdk`
3) Standalone app for Car UI (via cargadgetd)
4) Fully decentralized solution, based on multisig and federated servers


## Build instructions

Install [sfosbuild](https://github.com/MrCyjaneK/sfosbuild).

```
$ sfosbuild 5.1.0.11 aarch64 ./rpm/kiko-sfos.spec
```

rpms are in rpms/* directory.

> Hackathon entry for TUM Blockchain Conference in Munich.
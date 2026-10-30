import QtQuick 2.6
import Sailfish.Silica 1.0

Page {
    id: page
    property int index: -1
    property string host
    property string errorText
    property int coin: -1
    property string whole
    property string atom
    property string ticker
    property string feeUnit: "ada"
    property bool euroTried
    property string baseEurLabel
    property string perKmEurLabel
    property string baseQuoteMessage
    property string perKmQuoteMessage
    property bool compatible: true

    Component.onCompleted: load()

    function load() {
        if (index < 0)
            return
        hostField.text = host
        coin = kiko.serverCoin(index)
        compatible = kiko.serverCompatible(index) === 1
        if (coin >= 0) {
            whole = kiko.walletWhole(coin)
            atom = kiko.walletAtom(coin)
            ticker = kiko.walletTicker(coin)
            feeUnit = whole
        }
        baseCrypto.text = fieldFromLovelace(kiko.serverFeeBase(index))
        perKmCrypto.text = fieldFromLovelace(kiko.serverFeePerKm(index))
        if (kiko.lastError() !== "")
            errorText = kiko.lastError()
    }

    function stripTicker(text) {
        var suffix = " " + ticker
        if (ticker !== "" && text.length >= suffix.length && text.slice(-suffix.length) === suffix)
            return text.slice(0, text.length - suffix.length)
        return text
    }

    function fieldFromLovelace(lovelace) {
        if (lovelace === "" || coin < 0)
            return lovelace
        var parts = kiko.walletQuote(coin, lovelace, atom).split("\n")
        if (parts.length > 3 && parts[3] !== "")
            return lovelace
        if (feeUnit === atom)
            return parts[1]
        return stripTicker(parts[0])
    }

    function converted(text, fromUnit, toUnit) {
        if (text.trim() === "" || coin < 0)
            return text
        var parts = kiko.walletQuote(coin, text, fromUnit).split("\n")
        if (parts.length > 3 && parts[3] !== "")
            return text
        if (toUnit === atom)
            return parts[1]
        return stripTicker(parts[0])
    }

    function quoteInto(text, unit, which) {
        var eur = ""
        var message = ""
        if (text.trim() !== "" && coin >= 0) {
            var parts = kiko.walletQuote(coin, text, unit).split("\n")
            message = parts.length > 3 ? parts[3] : ""
            if (message === "")
                eur = parts[2]
            var lovelace = parts.length > 1 ? parts[1] : ""
            if (message === "" && eur === "" && lovelace !== "" && lovelace !== "0" && !euroTried) {
                euroTried = true
                kiko.walletWarmEuro(coin)
            }
        }
        if (which === "base") {
            baseQuoteMessage = message
            baseEurLabel = eur
        } else {
            perKmQuoteMessage = message
            perKmEurLabel = eur
        }
    }

    function refreshQuotes() {
        if (!baseCrypto)
            return
        quoteInto(baseCrypto.text, feeUnit, "base")
        quoteInto(perKmCrypto.text, feeUnit, "perKm")
    }

    Connections {
        target: kiko
        onEuroReady: {
            if (wallet === page.coin)
                page.refreshQuotes()
        }
    }

    SilicaFlickable {
        anchors.fill: parent
        contentHeight: column.height

        Column {
            id: column
            width: parent.width
            spacing: Theme.paddingLarge

            PageHeader { title: page.index < 0 ? "Add server" : "Server" }

            Label {
                visible: page.errorText !== ""
                width: parent.width - 2 * Theme.horizontalPageMargin
                x: Theme.horizontalPageMargin
                wrapMode: Text.Wrap
                color: Theme.highlightColor
                text: page.errorText
            }

            TextField {
                id: hostField
                width: parent.width
                label: "Host"
                placeholderText: "example.com"
                inputMethodHints: Qt.ImhNoAutoUppercase | Qt.ImhUrlCharactersOnly
            }

            Label {
                visible: page.index >= 0 && !page.compatible
                width: parent.width - 2 * Theme.horizontalPageMargin
                x: Theme.horizontalPageMargin
                wrapMode: Text.Wrap
                text: "No matching wallet for this server"
                color: Theme.highlightColor
                font.pixelSize: Theme.fontSizeSmall
            }

            Column {
                width: parent.width
                spacing: Theme.paddingLarge
                visible: page.index >= 0

                SectionHeader { text: "Price" }

                TextSwitch {
                    id: atomSwitch
                    width: parent.width
                    text: page.atom.charAt(0).toUpperCase() + page.atom.slice(1)
                    description: checked ? "Amounts in " + page.atom : "Amounts in " + page.ticker
                    onCheckedChanged: {
                        var next = checked ? page.atom : page.whole
                        if (next === "" || next === page.feeUnit)
                            return
                        var base = page.converted(baseCrypto.text, page.feeUnit, next)
                        var perKm = page.converted(perKmCrypto.text, page.feeUnit, next)
                        page.feeUnit = next
                        baseCrypto.text = base
                        perKmCrypto.text = perKm
                    }
                }

                TextField {
                    id: baseCrypto
                    width: parent.width
                    label: "Base fee"
                    placeholderText: page.feeUnit === page.atom ? atomSwitch.text : page.ticker
                    inputMethodHints: page.feeUnit === page.atom ? Qt.ImhDigitsOnly : Qt.ImhFormattedNumbersOnly
                    onTextChanged: page.quoteInto(text, page.feeUnit, "base")
                }

                Label {
                    visible: page.baseEurLabel !== ""
                    x: Theme.horizontalPageMargin
                    text: page.baseEurLabel
                    color: Theme.secondaryColor
                    font.pixelSize: Theme.fontSizeLarge
                }

                Label {
                    visible: page.baseQuoteMessage !== ""
                    width: parent.width - 2 * Theme.horizontalPageMargin
                    x: Theme.horizontalPageMargin
                    wrapMode: Text.Wrap
                    color: Theme.highlightColor
                    text: page.baseQuoteMessage
                }

                TextField {
                    id: perKmCrypto
                    width: parent.width
                    label: "Per km"
                    placeholderText: page.feeUnit === page.atom ? atomSwitch.text : page.ticker
                    inputMethodHints: page.feeUnit === page.atom ? Qt.ImhDigitsOnly : Qt.ImhFormattedNumbersOnly
                    onTextChanged: page.quoteInto(text, page.feeUnit, "perKm")
                }

                Label {
                    visible: page.perKmEurLabel !== ""
                    x: Theme.horizontalPageMargin
                    text: page.perKmEurLabel
                    color: Theme.secondaryColor
                    font.pixelSize: Theme.fontSizeLarge
                }

                Label {
                    visible: page.perKmQuoteMessage !== ""
                    width: parent.width - 2 * Theme.horizontalPageMargin
                    x: Theme.horizontalPageMargin
                    wrapMode: Text.Wrap
                    color: Theme.highlightColor
                    text: page.perKmQuoteMessage
                }
            }

            Button {
                anchors.horizontalCenter: parent.horizontalCenter
                text: page.index < 0 ? "Add" : "Save"
                onClicked: {
                    var rc = page.index < 0
                            ? kiko.serverAdd(hostField.text)
                            : kiko.serverUpdate(page.index, hostField.text)
                    if (rc !== 0) {
                        page.errorText = kiko.lastError()
                        return
                    }
                    if (page.index >= 0) {
                        rc = kiko.serverSaveFees(page.index, baseCrypto.text, perKmCrypto.text, page.feeUnit)
                        if (rc !== 0) {
                            page.errorText = kiko.lastError()
                            return
                        }
                    }
                    pageStack.pop()
                }
            }

            Button {
                visible: page.index >= 0
                anchors.horizontalCenter: parent.horizontalCenter
                text: "Delete"
                onClicked: {
                    if (kiko.serverDelete(page.index) === 0)
                        pageStack.pop()
                    else
                        page.errorText = kiko.lastError()
                }
            }
        }
    }
}

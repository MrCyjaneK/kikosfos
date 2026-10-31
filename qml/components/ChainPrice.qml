import QtQuick 2.6
import Sailfish.Silica 1.0

Column {
    id: root
    property string currency
    property int coin: -1
    property string coinName
    property string whole
    property string atom
    property string ticker
    property bool hasWallet
    property string baseFee
    property string perKmFee
    property string feeUnit
    property bool loading
    property string baseEurLabel
    property string perKmEurLabel
    property string baseQuoteMessage
    property string perKmQuoteMessage

    width: parent.width
    spacing: Theme.paddingLarge

    function loadFields() {
        if (!hasWallet || !baseCrypto || !atomSwitch)
            return
        loading = true
        feeUnit = whole
        atomSwitch.checked = false
        baseCrypto.text = fieldFromAtoms(baseFee)
        perKmCrypto.text = fieldFromAtoms(perKmFee)
        loading = false
        refreshQuotes()
    }

    function stripTicker(text) {
        var suffix = " " + ticker
        if (ticker !== "" && text.length >= suffix.length && text.slice(-suffix.length) === suffix)
            return text.slice(0, text.length - suffix.length)
        return text
    }

    function fieldFromAtoms(atoms) {
        if (atoms === "" || coin < 0)
            return atoms
        var parts = kiko.walletQuote(coin, atoms, atom).split("\n")
        if (parts.length > 3 && parts[3] !== "")
            return atoms
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

    function feePayload() {
        if (!hasWallet)
            return null
        return {
            currency: currency,
            base: baseCrypto.text,
            per_km: perKmCrypto.text,
            unit: feeUnit
        }
    }

    Component.onCompleted: loadFields()
    onBaseFeeChanged: loadFields()
    onCurrencyChanged: loadFields()

    Connections {
        target: kiko
        onEuroReady: {
            if (wallet === root.coin)
                root.refreshQuotes()
        }
    }

    SectionHeader { text: root.coinName }

    Label {
        visible: !root.hasWallet
        width: parent.width - 2 * Theme.horizontalPageMargin
        x: Theme.horizontalPageMargin
        wrapMode: Text.Wrap
        text: "No " + root.coinName + " wallet"
        color: Theme.highlightColor
        font.pixelSize: Theme.fontSizeSmall
    }

    TextSwitch {
        id: atomSwitch
        visible: root.hasWallet
        width: parent.width
        text: root.atom.charAt(0).toUpperCase() + root.atom.slice(1)
        description: checked ? "Amounts in " + root.atom : "Amounts in " + root.ticker
        onCheckedChanged: {
            if (root.loading)
                return
            var next = checked ? root.atom : root.whole
            if (next === "" || next === root.feeUnit)
                return
            var base = root.converted(baseCrypto.text, root.feeUnit, next)
            var perKm = root.converted(perKmCrypto.text, root.feeUnit, next)
            root.feeUnit = next
            baseCrypto.text = base
            perKmCrypto.text = perKm
        }
    }

    TextField {
        id: baseCrypto
        visible: root.hasWallet
        width: parent.width
        label: "Base fee"
        placeholderText: root.feeUnit === root.atom ? atomSwitch.text : root.ticker
        inputMethodHints: root.feeUnit === root.atom ? Qt.ImhDigitsOnly : Qt.ImhFormattedNumbersOnly
        onTextChanged: root.quoteInto(text, root.feeUnit, "base")
    }

    Label {
        visible: root.hasWallet && root.baseEurLabel !== ""
        x: Theme.horizontalPageMargin
        text: root.baseEurLabel
        color: Theme.secondaryColor
        font.pixelSize: Theme.fontSizeLarge
    }

    Label {
        visible: root.hasWallet && root.baseQuoteMessage !== ""
        width: parent.width - 2 * Theme.horizontalPageMargin
        x: Theme.horizontalPageMargin
        wrapMode: Text.Wrap
        color: Theme.highlightColor
        text: root.baseQuoteMessage
    }

    TextField {
        id: perKmCrypto
        visible: root.hasWallet
        width: parent.width
        label: "Per km"
        placeholderText: root.feeUnit === root.atom ? atomSwitch.text : root.ticker
        inputMethodHints: root.feeUnit === root.atom ? Qt.ImhDigitsOnly : Qt.ImhFormattedNumbersOnly
        onTextChanged: root.quoteInto(text, root.feeUnit, "perKm")
    }

    Label {
        visible: root.hasWallet && root.perKmEurLabel !== ""
        x: Theme.horizontalPageMargin
        text: root.perKmEurLabel
        color: Theme.secondaryColor
        font.pixelSize: Theme.fontSizeLarge
    }

    Label {
        visible: root.hasWallet && root.perKmQuoteMessage !== ""
        width: parent.width - 2 * Theme.horizontalPageMargin
        x: Theme.horizontalPageMargin
        wrapMode: Text.Wrap
        color: Theme.highlightColor
        text: root.perKmQuoteMessage
    }
}

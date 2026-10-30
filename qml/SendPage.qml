import QtQuick 2.6
import Sailfish.Silica 1.0

Page {
    id: page
    property int wallet
    property var coinPage
    property string whole: kiko.walletWhole(wallet)
    property string atom: kiko.walletAtom(wallet)
    property string ticker: kiko.walletTicker(wallet)
    property string unit: atomSwitch.checked ? atom : whole
    property string ada
    property string lovelace
    property string eur
    property string previewMessage
    property bool euroTried

    function refreshQuote() {
        if (!amountField)
            return
        if (amountField.text.trim() === "") {
            ada = ""
            lovelace = ""
            eur = ""
            previewMessage = ""
            return
        }
        var parts = kiko.walletQuote(wallet, amountField.text, unit).split("\n")
        var message = parts.length > 3 ? parts[3] : ""
        previewMessage = message
        if (message !== "") {
            ada = ""
            lovelace = ""
            eur = ""
            return
        }
        ada = parts[0]
        lovelace = parts[1]
        eur = parts[2]
        if (eur === "" && lovelace !== "" && lovelace !== "0" && !euroTried) {
            euroTried = true
            kiko.walletWarmEuro(wallet)
        }
    }

    Connections {
        target: kiko
        onEuroReady: {
            if (wallet === page.wallet)
                page.refreshQuote()
        }
    }

    SilicaFlickable {
        anchors.fill: parent
        contentHeight: Math.max(column.height, height)

        Column {
            id: column
            width: parent.width
            spacing: Theme.paddingLarge

            PageHeader { title: "Send" }

            TextField {
                id: recipient
                width: parent.width
                placeholderText: "Address"
            }

            TextSwitch {
                id: atomSwitch
                width: parent.width
                text: page.atom.charAt(0).toUpperCase() + page.atom.slice(1)
                description: checked ? "Amount in " + page.atom : "Amount in " + page.ticker
                onCheckedChanged: page.refreshQuote()
            }

            TextField {
                id: amountField
                width: parent.width
                placeholderText: page.unit === page.atom ? atomSwitch.text : page.ticker
                inputMethodHints: page.unit === page.atom ? Qt.ImhDigitsOnly : Qt.ImhFormattedNumbersOnly
                onTextChanged: page.refreshQuote()
            }

            Column {
                width: parent.width
                spacing: Theme.paddingSmall
                visible: page.ada !== "" || page.previewMessage !== ""

                Label {
                    anchors.horizontalCenter: parent.horizontalCenter
                    visible: page.ada !== ""
                    text: page.ada
                    color: Theme.primaryColor
                    font.pixelSize: Theme.fontSizeExtraLarge
                }

                Label {
                    anchors.horizontalCenter: parent.horizontalCenter
                    visible: page.lovelace !== ""
                    text: page.lovelace + " " + page.atom
                    color: Theme.secondaryColor
                    font.pixelSize: Theme.fontSizeMedium
                }

                Label {
                    anchors.horizontalCenter: parent.horizontalCenter
                    visible: page.eur !== ""
                    text: page.eur
                    color: Theme.secondaryColor
                    font.pixelSize: Theme.fontSizeLarge
                }

                Label {
                    visible: page.previewMessage !== ""
                    width: parent.width - 2 * Theme.horizontalPageMargin
                    x: Theme.horizontalPageMargin
                    wrapMode: Text.Wrap
                    color: Theme.highlightColor
                    text: page.previewMessage
                }
            }

            Button {
                anchors.horizontalCenter: parent.horizontalCenter
                text: "Review"
                enabled: page.lovelace !== "" && page.lovelace !== "0" && recipient.text.trim() !== "" && page.previewMessage === ""
                onClicked: pageStack.push(Qt.resolvedUrl("ConfirmSendPage.qml"), {
                    wallet: page.wallet,
                    coinPage: page.coinPage,
                    recipient: recipient.text.trim(),
                    ada: page.ada,
                    lovelace: page.lovelace,
                    atom: page.atom,
                    eur: page.eur
                })
            }
        }
    }

}

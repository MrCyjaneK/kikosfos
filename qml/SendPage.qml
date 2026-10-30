import QtQuick 2.6
import Sailfish.Silica 1.0

Page {
    id: page
    property int wallet
    property var coinPage
    property string unit: lovelaceSwitch.checked ? "lovelace" : "ada"
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
                id: lovelaceSwitch
                width: parent.width
                text: "Lovelace"
                description: checked ? "Amount in lovelace" : "Amount in ADA"
                onCheckedChanged: page.refreshQuote()
            }

            TextField {
                id: amountField
                width: parent.width
                placeholderText: page.unit === "lovelace" ? "Lovelace" : "ADA"
                inputMethodHints: page.unit === "lovelace" ? Qt.ImhDigitsOnly : Qt.ImhFormattedNumbersOnly
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
                    text: page.lovelace + " lovelace"
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
                    eur: page.eur
                })
            }
        }
    }

}

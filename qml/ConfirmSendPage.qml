import QtQuick 2.6
import Sailfish.Silica 1.0

Page {
    id: page
    property int wallet
    property var coinPage
    property string recipient
    property string ada
    property string lovelace
    property string atom
    property string eur
    property bool sending
    property string errorText

    Connections {
        target: kiko
        onSendFinished: {
            if (wallet !== page.wallet || !page.sending)
                return
            if (message === "") {
                pageStack.pop(page.coinPage)
                return
            }
            page.sending = false
            page.errorText = message
        }
    }

    SilicaFlickable {
        anchors.fill: parent
        contentHeight: Math.max(column.height, height)

        Column {
            id: column
            width: parent.width
            spacing: Theme.paddingLarge

            PageHeader { title: "Confirm" }

            Label {
                anchors.horizontalCenter: parent.horizontalCenter
                text: "To"
                color: Theme.secondaryColor
                font.pixelSize: Theme.fontSizeSmall
            }

            Label {
                x: Theme.horizontalPageMargin
                width: parent.width - 2 * Theme.horizontalPageMargin
                wrapMode: Text.WrapAnywhere
                horizontalAlignment: Text.AlignHCenter
                text: page.recipient
                color: Theme.primaryColor
                font.pixelSize: Theme.fontSizeSmall
            }

            Label {
                anchors.horizontalCenter: parent.horizontalCenter
                text: page.ada
                color: Theme.primaryColor
                font.pixelSize: Theme.fontSizeExtraLarge
            }

            Label {
                anchors.horizontalCenter: parent.horizontalCenter
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
                x: Theme.horizontalPageMargin
                width: parent.width - 2 * Theme.horizontalPageMargin
                wrapMode: Text.Wrap
                horizontalAlignment: Text.AlignHCenter
                text: "The network fee comes out of this wallet."
                color: Theme.secondaryColor
                font.pixelSize: Theme.fontSizeSmall
            }

            BusyIndicator {
                anchors.horizontalCenter: parent.horizontalCenter
                running: page.sending
                visible: running
                width: Theme.itemSizeSmall
                height: width
            }

            Label {
                visible: page.errorText !== ""
                width: parent.width - 2 * Theme.horizontalPageMargin
                x: Theme.horizontalPageMargin
                wrapMode: Text.Wrap
                color: Theme.highlightColor
                text: page.errorText
            }

            Button {
                anchors.horizontalCenter: parent.horizontalCenter
                text: "Send"
                enabled: !page.sending
                onClicked: {
                    page.errorText = ""
                    page.sending = true
                    kiko.walletSend(page.wallet, page.recipient, page.lovelace)
                }
            }
        }
    }
}

import QtQuick 2.6
import Sailfish.Silica 1.0
import "components"

Page {
    id: page
    property int wallet
    property var coinPage
    property string pin
    property string errorText

    function bullets(n) {
        var s = ""
        for (var i = 0; i < n; ++i)
            s += "•"
        return s
    }

    function submit() {
        if (pin.length === 0)
            return
        if (kiko.walletDelete(wallet, pin) === 0) {
            if (coinPage)
                pageStack.pop(coinPage)
            else
                pageStack.pop()
        } else {
            errorText = kiko.lastError()
            pin = ""
        }
    }

    SilicaFlickable {
        anchors.top: parent.top
        anchors.left: parent.left
        anchors.right: parent.right
        anchors.bottom: pad.top
        contentHeight: column.height

        Column {
            id: column
            width: parent.width
            spacing: Theme.paddingLarge

            PageHeader { title: "Confirm PIN" }

            Label {
                width: parent.width - 2 * Theme.horizontalPageMargin
                anchors.horizontalCenter: parent.horizontalCenter
                horizontalAlignment: Text.AlignHCenter
                wrapMode: Text.Wrap
                color: Theme.highlightColor
                text: page.errorText
            }

            Label {
                anchors.horizontalCenter: parent.horizontalCenter
                font.pixelSize: Theme.fontSizeExtraLarge
                text: page.bullets(page.pin.length)
            }
        }
    }

    PinPad {
        id: pad
        anchors.bottom: parent.bottom
        onDigit: page.pin += text
        onBackspace: page.pin = page.pin.slice(0, -1)
        onAccept: page.submit()
    }
}

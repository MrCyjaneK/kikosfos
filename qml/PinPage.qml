import QtQuick 2.6
import Sailfish.Silica 1.0
import "components"

Page {
    id: page

    property string pin
    property string firstPin
    property bool confirming: false
    property bool busy: false
    property string errorText
    property int pinState: kikoInitError ? -1 : kiko.pinIsSet()

    function heading() {
        if (kikoInitError || pinState < 0)
            return "Wallet unavailable"
        if (pinState === 0 && !confirming)
            return "Choose a PIN"
        if (confirming)
            return "Repeat the PIN"
        return "Enter PIN"
    }

    function bullets(n) {
        var s = ""
        for (var i = 0; i < n; ++i)
            s += "•"
        return s
    }

    function submit() {
        if (busy || pin.length === 0 || pinState < 0)
            return
        errorText = ""
        if (pinState === 0 && !confirming) {
            firstPin = pin
            pin = ""
            confirming = true
            return
        }
        busy = true
        if (pinState === 0)
            kiko.setupPin(firstPin, pin)
        else
            kiko.unlockPin(pin)
    }

    Connections {
        target: kiko
        onPinAccepted: pageStack.replace(Qt.resolvedUrl("HomePage.qml"))
        onPinRejected: {
            page.busy = false
            page.pin = ""
            page.firstPin = ""
            page.confirming = false
            page.errorText = message
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

            PageHeader { title: page.heading() }

            Label {
                width: parent.width - 2 * Theme.horizontalPageMargin
                anchors.horizontalCenter: parent.horizontalCenter
                horizontalAlignment: Text.AlignHCenter
                wrapMode: Text.Wrap
                color: Theme.highlightColor
                text: kikoInitError ? kikoInitError : page.errorText
            }

            Label {
                anchors.horizontalCenter: parent.horizontalCenter
                font.pixelSize: Theme.fontSizeExtraLarge
                text: page.bullets(page.pin.length)
            }

            BusyIndicator {
                anchors.horizontalCenter: parent.horizontalCenter
                running: page.busy
                size: BusyIndicatorSize.Medium
            }
        }
    }

    PinPad {
        id: pad
        anchors.bottom: parent.bottom
        enabled: !page.busy && !kikoInitError && page.pinState >= 0
        onDigit: page.pin += text
        onBackspace: page.pin = page.pin.slice(0, -1)
        onAccept: page.submit()
    }
}

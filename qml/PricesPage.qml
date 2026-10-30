import QtQuick 2.6
import Sailfish.Silica 1.0

Page {
    id: page
    property string errorText
    property bool applying

    SilicaFlickable {
        anchors.fill: parent
        contentHeight: column.height

        Column {
            id: column
            width: parent.width
            spacing: Theme.paddingLarge

            PageHeader { title: "Set all prices" }

            Label {
                width: parent.width - 2 * Theme.horizontalPageMargin
                x: Theme.horizontalPageMargin
                wrapMode: Text.Wrap
                text: "Sets the base fee and the price per kilometre on every server, in that server's currency."
                color: Theme.secondaryColor
                font.pixelSize: Theme.fontSizeSmall
            }

            Label {
                visible: page.errorText !== ""
                width: parent.width - 2 * Theme.horizontalPageMargin
                x: Theme.horizontalPageMargin
                wrapMode: Text.Wrap
                color: Theme.highlightColor
                text: page.errorText
            }

            TextField {
                id: baseEur
                width: parent.width
                label: "Base fee"
                placeholderText: "EUR"
                inputMethodHints: Qt.ImhFormattedNumbersOnly
                EnterKey.iconSource: "image://theme/icon-m-enter-next"
                EnterKey.onClicked: perKmEur.focus = true
            }

            TextField {
                id: perKmEur
                width: parent.width
                label: "Per km"
                placeholderText: "EUR"
                inputMethodHints: Qt.ImhFormattedNumbersOnly
                EnterKey.iconSource: "image://theme/icon-m-enter-accept"
                EnterKey.onClicked: page.apply()
            }

            Button {
                anchors.horizontalCenter: parent.horizontalCenter
                text: "Update all servers"
                enabled: !page.applying
                onClicked: page.apply()
            }

            BusyIndicator {
                anchors.horizontalCenter: parent.horizontalCenter
                running: page.applying
                visible: page.applying
                width: Theme.itemSizeSmall
                height: width
            }
        }
    }

    function apply() {
        if (applying)
            return
        applying = true
        errorText = ""
        kiko.applyFeesEur(baseEur.text, perKmEur.text)
    }

    Connections {
        target: kiko
        onFeesApplied: {
            page.applying = false
            if (message !== "") {
                page.errorText = message
                return
            }
            pageStack.pop()
        }
    }
}

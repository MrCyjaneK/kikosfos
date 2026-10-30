import QtQuick 2.6
import Sailfish.Silica 1.0

Page {
    PullDownMenu {
        MenuItem {
            text: "Wallets"
            onClicked: pageStack.push(Qt.resolvedUrl("WalletsPage.qml"))
        }
    }

    PageHeader { title: "Kiko" }

    Label {
        anchors.centerIn: parent
        text: "Pull down for wallets"
        color: Theme.secondaryColor
    }
}

import QtQuick 2.6
import Sailfish.Silica 1.0

Page {
    SilicaFlickable {
        anchors.fill: parent
        contentHeight: Math.max(column.height, height)

        PullDownMenu {
            MenuItem {
                text: "Wallets"
                onClicked: pageStack.push(Qt.resolvedUrl("WalletsPage.qml"))
            }
            MenuItem {
                text: "Servers"
                onClicked: pageStack.push(Qt.resolvedUrl("ServersPage.qml"))
            }
            MenuItem {
                text: "Profile"
                onClicked: pageStack.push(Qt.resolvedUrl("ProfilePage.qml"))
            }
        }

        Column {
            id: column
            width: parent.width

            PageHeader { title: "Kiko" }

            Label {
                width: parent.width
                horizontalAlignment: Text.AlignHCenter
                text: "Pull down for wallets, servers, and profile"
                color: Theme.secondaryColor
            }
        }
    }
}

import QtQuick 2.6
import Sailfish.Silica 1.0

Page {
    SilicaListView {
        anchors.fill: parent
        header: PageHeader { title: "Wallets" }
        model: kiko.walletCount()

        delegate: ListItem {
            contentHeight: Theme.itemSizeMedium

            Image {
                id: icon
                anchors.left: parent.left
                anchors.leftMargin: Theme.horizontalPageMargin
                anchors.verticalCenter: parent.verticalCenter
                width: Theme.iconSizeMedium
                height: Theme.iconSizeMedium
                source: "image://kiko/icon/" + index
                fillMode: Image.PreserveAspectFit
            }

            Label {
                anchors.left: icon.right
                anchors.leftMargin: Theme.paddingMedium
                anchors.right: parent.right
                anchors.rightMargin: Theme.horizontalPageMargin
                anchors.verticalCenter: parent.verticalCenter
                text: kiko.walletName(index)
            }

            onClicked: pageStack.push(Qt.resolvedUrl("CoinPage.qml"), { wallet: index })
        }
    }
}

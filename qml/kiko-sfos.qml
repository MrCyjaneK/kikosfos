import QtQuick 2.6
import Sailfish.Silica 1.0

ApplicationWindow {
    initialPage: Component {
        Page {
            Label {
                anchors.centerIn: parent
                text: kiko.hello()
            }
        }
    }
}

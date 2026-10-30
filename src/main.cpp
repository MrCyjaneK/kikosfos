#include <QtQuick>
#include <sailfishapp.h>

#include "kikoapi.h"

int main(int argc, char *argv[])
{
    QScopedPointer<QGuiApplication> app(SailfishApp::application(argc, argv));
    QScopedPointer<QQuickView> view(SailfishApp::createView());

    view->rootContext()->setContextProperty(QStringLiteral("kiko"), new KikoApi(view.data()));
    view->setSource(SailfishApp::pathToMainQml());
    view->show();

    return app->exec();
}

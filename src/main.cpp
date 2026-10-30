#include <QtQuick>
#include <QDir>
#include <QStandardPaths>
#include <sailfishapp.h>

#include "kikoapi.h"
#include "kikoimage.h"
#include "../go/libkiko.h"

int main(int argc, char *argv[])
{
    QScopedPointer<QGuiApplication> app(SailfishApp::application(argc, argv));
    QCoreApplication::setOrganizationName(QStringLiteral("wtf.kiko"));
    QCoreApplication::setApplicationName(QStringLiteral("kiko"));

    const QString configDir = QStandardPaths::writableLocation(QStandardPaths::AppConfigLocation);
    const QString dataDir = QStandardPaths::writableLocation(QStandardPaths::AppDataLocation);
    QDir().mkpath(configDir);
    QDir().mkpath(dataDir);

    QByteArray configUtf8 = configDir.toUtf8();
    QByteArray dataUtf8 = dataDir.toUtf8();
    QString initError;
    if (KIKO_init(configUtf8.data(), dataUtf8.data()) != 0)
        initError = takeKikoString(KIKO_last_error());

    QScopedPointer<QQuickView> view(SailfishApp::createView());
    view->engine()->addImageProvider(QStringLiteral("kiko"), new KikoImageProvider);
    view->rootContext()->setContextProperty(QStringLiteral("kiko"), new KikoApi(view.data()));
    view->rootContext()->setContextProperty(QStringLiteral("kikoInitError"), initError);
    view->setSource(SailfishApp::pathToMainQml());
    view->show();

    return app->exec();
}

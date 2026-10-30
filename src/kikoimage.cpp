#include "kikoimage.h"

#include "../go/libkiko.h"

KikoImageProvider::KikoImageProvider()
    : QQuickImageProvider(QQuickImageProvider::Image)
{
}

QImage KikoImageProvider::requestImage(const QString &id, QSize *size, const QSize &requestedSize)
{
    Q_UNUSED(requestedSize)
    const int slash = id.indexOf(QLatin1Char('/'));
    if (slash <= 0) {
        if (size)
            *size = QSize();
        return QImage();
    }
    bool ok = false;
    const int index = id.mid(slash + 1).toInt(&ok);
    if (!ok) {
        if (size)
            *size = QSize();
        return QImage();
    }

    int len = 0;
    unsigned char *bytes = nullptr;
    const QString kind = id.left(slash);
    if (kind == QLatin1String("icon"))
        bytes = KIKO_wallet_icon(index, &len);
    else if (kind == QLatin1String("qr"))
        bytes = KIKO_wallet_qr(index, &len);

    QImage image;
    if (bytes && len > 0)
        image.loadFromData(bytes, len, "PNG");
    if (bytes)
        KIKO_free(bytes);
    if (size)
        *size = image.size();
    return image;
}

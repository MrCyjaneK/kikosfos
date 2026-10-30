#pragma once

#include <QQuickImageProvider>

class KikoImageProvider : public QQuickImageProvider
{
public:
    KikoImageProvider();

    QImage requestImage(const QString &id, QSize *size, const QSize &requestedSize) override;
};

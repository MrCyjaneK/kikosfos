Name:       kiko-sfos
Summary:    Kiko for Sailfish
Version:    0.1.0
Release:    1
License:    BSD-3-Clause
Source0:    %{name}-%{version}.tar.bz2
BuildRequires:  pkgconfig(Qt5Core)
BuildRequires:  pkgconfig(Qt5Gui)
BuildRequires:  pkgconfig(Qt5Qml)
BuildRequires:  pkgconfig(Qt5Quick)
BuildRequires:  pkgconfig(Qt5Concurrent)
BuildRequires:  pkgconfig(sailfishapp)
Requires:       sailfishsilica-qt5

%description
Kiko for Sailfish.

%prep
%setup -q -n %{name}-%{version}

%build
CGO_ENABLED=1 go build -C go -buildmode=c-shared -ldflags '-s -w' -o libkiko.so .
%qmake5
%make_build

%install
%qmake5_install

%files
%license LICENSES/BSD-3-Clause.txt
%{_bindir}/kiko-sfos
%{_datadir}/kiko-sfos/qml
%{_datadir}/applications/kiko-sfos.desktop
%{_libdir}/libkiko.so

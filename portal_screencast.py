#!/usr/bin/env python3
"""
XDG Desktop Portal üzerinden ekran yakalama başlatır ve PipeWire node_id'yi döndürür.
Hyprland/Wayland için ekran paylaşımı.
"""

import dbus
from dbus.mainloop.glib import DBusGMainLoop
from gi.repository import GLib
import sys
import random
import string

DBusGMainLoop(set_as_default=True)

def random_token():
    return ''.join(random.choices(string.ascii_lowercase + string.digits, k=16))

class ScreenCast:
    def __init__(self):
        self.bus = dbus.SessionBus()
        self.portal = self.bus.get_object(
            'org.freedesktop.portal.Desktop',
            '/org/freedesktop/portal/desktop'
        )
        self.screencast = dbus.Interface(
            self.portal,
            'org.freedesktop.portal.ScreenCast'
        )
        self.session_handle = None
        self.node_id = None
        self.loop = GLib.MainLoop()
        self.sender_name = self.bus.get_unique_name().replace('.', '_').replace(':', '')
        
    def handle_create_session_response(self, response, results):
        if response != 0:
            print(f"CreateSession hatası: {response}", file=sys.stderr)
            self.loop.quit()
            return
        self.session_handle = results.get('session_handle')
        print(f"Session oluşturuldu: {self.session_handle}", file=sys.stderr)
        self.select_sources()
        
    def handle_select_sources_response(self, response, results):
        if response != 0:
            print(f"SelectSources hatası: {response}", file=sys.stderr)
            self.loop.quit()
            return
        print("Kaynaklar seçildi", file=sys.stderr)
        self.start()
        
    def handle_start_response(self, response, results):
        if response != 0:
            print(f"Start hatası: {response}", file=sys.stderr)
            self.loop.quit()
            return
        streams = results.get('streams', [])
        if streams:
            self.node_id = str(streams[0][0])
            print(self.node_id)  # stdout'a node_id yaz
        else:
            print("Stream bulunamadı", file=sys.stderr)
        self.loop.quit()
        
    def create_session(self):
        token = random_token()
        request_token = random_token()
        request_path = f"/org/freedesktop/portal/desktop/request/{self.sender_name}/{request_token}"
        
        self.bus.add_signal_receiver(
            self.handle_create_session_response,
            signal_name='Response',
            dbus_interface='org.freedesktop.portal.Request',
            path=request_path
        )
        
        self.screencast.CreateSession({
            'session_handle_token': token,
            'handle_token': request_token
        })
        
    def select_sources(self):
        request_token = random_token()
        request_path = f"/org/freedesktop/portal/desktop/request/{self.sender_name}/{request_token}"
        
        self.bus.add_signal_receiver(
            self.handle_select_sources_response,
            signal_name='Response',
            dbus_interface='org.freedesktop.portal.Request',
            path=request_path
        )
        
        self.screencast.SelectSources(
            self.session_handle,
            {
                'types': dbus.UInt32(1),  # 1=Monitor
                'cursor_mode': dbus.UInt32(2),  # 2=Embedded
                'handle_token': request_token
            }
        )
        
    def start(self):
        request_token = random_token()
        request_path = f"/org/freedesktop/portal/desktop/request/{self.sender_name}/{request_token}"
        
        self.bus.add_signal_receiver(
            self.handle_start_response,
            signal_name='Response',
            dbus_interface='org.freedesktop.portal.Request',
            path=request_path
        )
        
        self.screencast.Start(
            self.session_handle,
            '',
            {
                'handle_token': request_token
            }
        )
        
    def run(self):
        self.create_session()
        GLib.timeout_add_seconds(60, self.timeout)  # 60 saniye timeout
        self.loop.run()
        return self.node_id
        
    def timeout(self):
        print("Timeout - ekran seçimi yapılmadı", file=sys.stderr)
        self.loop.quit()
        return False

if __name__ == '__main__':
    sc = ScreenCast()
    node_id = sc.run()
    if node_id is None:
        sys.exit(1)

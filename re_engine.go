package main

func getReverseEngineeringScript() string {
	return `
		// =========================================================================
		// ndWhatsApp: Advanced Reverse Engineering & Webpack Store Extraction Engine
		// =========================================================================
		(function() {
			if (window.__nd_re_initialized) return;
			window.__nd_re_initialized = true;

			// Global RE Namespace
			window.ndWA = {
				version: '1.0.0-re',
				ready: false,
				Store: {},
				rawModules: {},
				eventLog: [],
				eventListeners: { message: [] },

				getChats: function() {
					if (window.ndWA.Store.Chat && window.ndWA.Store.Chat.models) {
						return window.ndWA.Store.Chat.models;
					}
					return [];
				},

				getContacts: function() {
					if (window.ndWA.Store.Contact && window.ndWA.Store.Contact.models) {
						return window.ndWA.Store.Contact.models;
					}
					return [];
				},

				getConn: function() {
					return window.ndWA.Store.Conn || null;
				},

				onMessage: function(cb) {
					if (typeof cb === 'function') {
						window.ndWA.eventListeners.message.push(cb);
					}
				},

				inspectModule: function(query) {
					var results = [];
					var req = window.ndWA.require;
					if (!req || !req.m) return results;
					var q = (query || '').toLowerCase();
					for (var id in req.m) {
						try {
							var m = req(id);
							if (!m) continue;
							var keys = Object.keys(m).join(' ');
							if (m.default) keys += ' ' + Object.keys(m.default).join(' ');
							if (keys.toLowerCase().indexOf(q) !== -1 || id.toString().indexOf(q) !== -1) {
								results.push({ id: id, exports: m });
							}
						} catch (e) {}
					}
					return results;
				},

				toggleHUD: function() {
					var hud = document.getElementById('nd-re-hud');
					if (!hud) {
						injectDevHUD();
						hud = document.getElementById('nd-re-hud');
					}
					if (hud) {
						var isHidden = hud.style.display === 'none';
						hud.style.display = isHidden ? 'flex' : 'none';
						if (isHidden && window.ndUpdateHUD) {
							window.ndUpdateHUD();
						}
					}
				}
			};

			// Alias for standard compatibility with existing tools
			window.WA = window.ndWA;

			// Hook into WhatsApp Webpack chunk loader
			function hookWebpack() {
				if (!window.webpackChunkwhatsapp_web_client) return false;

				var probeId = '__nd_probe_' + Date.now();
				try {
					window.webpackChunkwhatsapp_web_client.push([
						[probeId],
						{},
						function(require) {
							window.ndWA.require = require;
							var modules = require.m || {};
							window.ndWA.rawModules = modules;

							for (var id in modules) {
								try {
									var m = require(id);
									if (!m) continue;

									// Msg module
									if (!window.ndWA.Store.Msg) {
										if (m.Msg) window.ndWA.Store.Msg = m.Msg;
										else if (m.default && m.default.Msg) window.ndWA.Store.Msg = m.default.Msg;
										else if (m.models && m.add && m.get) window.ndWA.Store.Msg = m;
									}

									// Chat module
									if (!window.ndWA.Store.Chat) {
										if (m.Chat) window.ndWA.Store.Chat = m.Chat;
										else if (m.default && m.default.Chat) window.ndWA.Store.Chat = m.default.Chat;
									}

									// Contact module
									if (!window.ndWA.Store.Contact) {
										if (m.Contact) window.ndWA.Store.Contact = m.Contact;
										else if (m.default && m.default.Contact) window.ndWA.Store.Contact = m.default.Contact;
									}

									// Conn module
									if (!window.ndWA.Store.Conn) {
										if (m.Conn) window.ndWA.Store.Conn = m.Conn;
										else if (m.default && m.default.Conn) window.ndWA.Store.Conn = m.default.Conn;
									}

									// Socket module
									if (!window.ndWA.Store.Socket) {
										if (m.Socket) window.ndWA.Store.Socket = m.Socket;
										else if (m.default && m.default.Socket) window.ndWA.Store.Socket = m.default.Socket;
									}

									// User module
									if (!window.ndWA.Store.User) {
										if (m.User) window.ndWA.Store.User = m.User;
										else if (m.default && m.default.User) window.ndWA.Store.User = m.default.User;
									}

									// Cmd module
									if (!window.ndWA.Store.Cmd) {
										if (m.Cmd) window.ndWA.Store.Cmd = m.Cmd;
										else if (m.default && m.default.Cmd) window.ndWA.Store.Cmd = m.default.Cmd;
									}
								} catch (err) {}
							}

							window.ndWA.ready = true;
							console.log('🔬 [ndWhatsApp] Webpack hooked! Modules count:', Object.keys(modules).length);
							console.log('🔬 [ndWhatsApp] Access window.ndWA or window.WA in console');

							// Attach message logger if Msg is hooked
							if (window.ndWA.Store.Msg && typeof window.ndWA.Store.Msg.on === 'function') {
								window.ndWA.Store.Msg.on('add', function(msg) {
									var entry = {
										time: new Date().toLocaleTimeString(),
										id: msg.id ? (msg.id._serialized || msg.id.id || '') : '',
										from: msg.from ? (msg.from._serialized || msg.from) : '',
										type: msg.type || 'unknown',
										body: (msg.body || msg.caption || '').slice(0, 80)
									};
									window.ndWA.eventLog.unshift(entry);
									if (window.ndWA.eventLog.length > 100) window.ndWA.eventLog.pop();

									var list = window.ndWA.eventListeners.message;
									for (var l = 0; l < list.length; l++) {
										try { list[l](msg); } catch(e){}
									}

									if (window.ndUpdateLiveEvents) {
										window.ndUpdateLiveEvents();
									}
								});
							}

							if (window.ndUpdateHUD) {
								window.ndUpdateHUD();
							}
						}
					]);
					return true;
				} catch (e) {
					return false;
				}
			}

			var hookInterval = setInterval(function() {
				if (hookWebpack()) {
					clearInterval(hookInterval);
				}
			}, 800);

			// Unblock Native Context Menu with Shift + Right Click
			window.addEventListener('contextmenu', function(e) {
				if (e.shiftKey) {
					e.stopPropagation();
				}
			}, true);

			// Keyboard shortcuts:
			// F12 or Ctrl + Shift + D -> Toggle DevTools HUD
			window.addEventListener('keydown', function(e) {
				if (e.key === 'F12' || ((e.ctrlKey || e.metaKey) && e.shiftKey && (e.key === 'd' || e.key === 'D'))) {
					e.preventDefault();
					window.ndWA.toggleHUD();
				}
			});

			// In-App DevTools HUD UI Component
			function injectDevHUD() {
				if (document.getElementById('nd-re-hud')) return;

				var hud = document.createElement('div');
				hud.id = 'nd-re-hud';
				hud.style.cssText = 'position:fixed;bottom:16px;right:16px;width:440px;height:520px;max-height:85vh;background:#111b21;color:#e9edef;z-index:999999;border-radius:12px;box-shadow:0 12px 40px rgba(0,0,0,0.75);border:1px solid #2a3942;display:none;flex-direction:column;font-family:-apple-system,BlinkMacSystemFont,"Segoe UI",Roboto,Helvetica,Arial,sans-serif;font-size:12px;overflow:hidden;backdrop-filter:blur(10px);';

				hud.innerHTML = '' +
					'<div style="display:flex;align-items:center;justify-content:space-between;padding:10px 14px;background:#202c33;border-bottom:1px solid #2a3942;user-select:none;">' +
					'  <div style="display:flex;align-items:center;gap:8px;">' +
					'    <span style="font-size:15px;">🔬</span>' +
					'    <strong style="font-size:12.5px;color:#00a884;letter-spacing:0.3px;">ndWhatsApp DevTools & RE</strong>' +
					'    <span id="nd-hud-badge-status" style="font-size:9.5px;padding:1px 6px;border-radius:10px;background:#374045;color:#8696a0;">Initializing...</span>' +
					'  </div>' +
					'  <div style="display:flex;align-items:center;gap:6px;">' +
					'    <button id="nd-hud-btn-close" style="background:transparent;border:none;color:#8696a0;cursor:pointer;font-size:16px;line-height:1;padding:2px 4px;">✕</button>' +
					'  </div>' +
					'</div>' +
					'<div style="display:flex;background:#182229;border-bottom:1px solid #2a3942;font-size:11px;">' +
					'  <button class="nd-tab-btn active" data-tab="metrics" style="flex:1;padding:8px 4px;background:#202c33;border:none;color:#00a884;cursor:pointer;font-weight:600;border-bottom:2px solid #00a884;">⚡ Metrics</button>' +
					'  <button class="nd-tab-btn" data-tab="store" style="flex:1;padding:8px 4px;background:transparent;border:none;color:#8696a0;cursor:pointer;font-weight:600;border-bottom:2px solid transparent;">💬 Store</button>' +
					'  <button class="nd-tab-btn" data-tab="events" style="flex:1;padding:8px 4px;background:transparent;border:none;color:#8696a0;cursor:pointer;font-weight:600;border-bottom:2px solid transparent;">📡 Events</button>' +
					'  <button class="nd-tab-btn" data-tab="runner" style="flex:1;padding:8px 4px;background:transparent;border:none;color:#8696a0;cursor:pointer;font-weight:600;border-bottom:2px solid transparent;">💻 Run JS</button>' +
					'</div>' +
					'<div id="nd-hud-body" style="flex:1;overflow-y:auto;padding:12px;display:flex;flex-direction:column;gap:10px;">' +
					'  <!-- Tab Metrics -->' +
					'  <div id="nd-tab-metrics" class="nd-tab-content">' +
					'    <div style="background:#202c33;border-radius:8px;padding:10px;border:1px solid #2a3942;margin-bottom:8px;">' +
					'      <div style="font-size:11px;color:#8696a0;margin-bottom:4px;text-transform:uppercase;font-weight:600;">Process & Runtime</div>' +
					'      <div style="display:grid;grid-template-columns:1fr 1fr;gap:6px;font-family:monospace;font-size:11px;">' +
					'        <div>PID: <span id="nd-stat-pid" style="color:#00a884;">-</span></div>' +
					'        <div>Heap Alloc: <span id="nd-stat-alloc" style="color:#00a884;">-</span></div>' +
					'        <div>Goroutines: <span id="nd-stat-goroutines" style="color:#00a884;">-</span></div>' +
					'        <div>Sys Mem: <span id="nd-stat-sys" style="color:#00a884;">-</span></div>' +
					'      </div>' +
					'    </div>' +
					'    <div style="background:#202c33;border-radius:8px;padding:10px;border:1px solid #2a3942;margin-bottom:8px;">' +
					'      <div style="font-size:11px;color:#8696a0;margin-bottom:4px;text-transform:uppercase;font-weight:600;">Remote WebKit Inspector</div>' +
					'      <div style="font-size:11px;line-height:1.4;margin-bottom:6px;">Buka peramban Chromium/Chrome Anda untuk inspect visual lengkap:</div>' +
					'      <div style="display:flex;align-items:center;gap:6px;">' +
					'        <code style="background:#111b21;padding:4px 8px;border-radius:4px;color:#53bdeb;flex:1;font-size:10.5px;">http://127.0.0.1:9222</code>' +
					'        <button id="nd-btn-copy-insp" style="background:#00a884;color:#111b21;border:none;border-radius:4px;padding:4px 8px;cursor:pointer;font-weight:600;font-size:10.5px;">Copy</button>' +
					'      </div>' +
					'    </div>' +
					'    <div style="background:#202c33;border-radius:8px;padding:10px;border:1px solid #2a3942;">' +
					'      <div style="font-size:11px;color:#8696a0;margin-bottom:4px;text-transform:uppercase;font-weight:600;">Shortcuts & Actions</div>' +
					'      <div style="font-size:11px;display:flex;flex-direction:column;gap:4px;">' +
					'        <div>• <kbd style="background:#111b21;padding:1px 4px;border-radius:3px;">Shift + Klik Kanan</kbd> : Munculkan menu Inspect Element</div>' +
					'        <div>• <kbd style="background:#111b21;padding:1px 4px;border-radius:3px;">F12</kbd> / <kbd style="background:#111b21;padding:1px 4px;border-radius:3px;">Ctrl+Shift+D</kbd> : Toggle HUD ini</div>' +
					'        <div>• <kbd style="background:#111b21;padding:1px 4px;border-radius:3px;">Ctrl + [ / ]</kbd> : Resize Sidebar Chat</div>' +
					'      </div>' +
					'    </div>' +
					'  </div>' +
					'  <!-- Tab Store -->' +
					'  <div id="nd-tab-store" class="nd-tab-content" style="display:none;">' +
					'    <div style="display:flex;justify-content:space-between;align-items:center;margin-bottom:6px;">' +
					'      <span style="font-size:11px;color:#8696a0;">Chat Models: <strong id="nd-stat-chats-count" style="color:#00a884;">0</strong></span>' +
					'      <button id="nd-btn-log-store" style="background:#202c33;border:1px solid #2a3942;color:#e9edef;padding:3px 8px;border-radius:4px;cursor:pointer;font-size:10.5px;">Log window.WA to Console</button>' +
					'    </div>' +
					'    <div id="nd-chat-list-preview" style="max-height:300px;overflow-y:auto;background:#111b21;border-radius:6px;padding:6px;border:1px solid #2a3942;font-family:monospace;font-size:10.5px;display:flex;flex-direction:column;gap:4px;">' +
					'      <div style="color:#8696a0;text-align:center;padding:10px;">Menunggu obrolan dimuat...</div>' +
					'    </div>' +
					'  </div>' +
					'  <!-- Tab Events -->' +
					'  <div id="nd-tab-events" class="nd-tab-content" style="display:none;">' +
					'    <div style="display:flex;justify-content:space-between;align-items:center;margin-bottom:6px;">' +
					'      <span style="font-size:11px;color:#8696a0;">Live Incoming Messages:</span>' +
					'      <button id="nd-btn-clear-events" style="background:#202c33;border:1px solid #2a3942;color:#8696a0;padding:2px 6px;border-radius:4px;cursor:pointer;font-size:10px;">Clear</button>' +
					'    </div>' +
					'    <div id="nd-event-log-container" style="max-height:320px;overflow-y:auto;background:#111b21;border-radius:6px;padding:6px;border:1px solid #2a3942;font-family:monospace;font-size:10px;display:flex;flex-direction:column;gap:4px;">' +
					'      <div style="color:#8696a0;text-align:center;padding:10px;">Menunggu pesan masuk...</div>' +
					'    </div>' +
					'  </div>' +
					'  <!-- Tab Runner -->' +
					'  <div id="nd-tab-runner" class="nd-tab-content" style="display:none;">' +
					'    <div style="margin-bottom:6px;font-size:11px;color:#8696a0;">Evaluate JS Expression in Context:</div>' +
					'    <div style="display:flex;gap:6px;margin-bottom:8px;">' +
					'      <input id="nd-input-eval" type="text" placeholder="e.g. ndWA.getChats().length" style="flex:1;background:#111b21;border:1px solid #2a3942;color:#e9edef;padding:6px 10px;border-radius:6px;font-family:monospace;font-size:11px;" />' +
					'      <button id="nd-btn-run-eval" style="background:#00a884;color:#111b21;border:none;border-radius:6px;padding:6px 12px;font-weight:600;cursor:pointer;">Run</button>' +
					'    </div>' +
					'    <div id="nd-eval-result" style="max-height:260px;overflow-y:auto;background:#111b21;border:1px solid #2a3942;border-radius:6px;padding:8px;font-family:monospace;font-size:10.5px;color:#00a884;white-space:pre-wrap;word-break:break-all;">Ready. Ketik ekspresi di atas lalu klik Run.</div>' +
					'  </div>' +
					'</div>';

				document.body.appendChild(hud);

				// Bind UI actions
				document.getElementById('nd-hud-btn-close').onclick = function() {
					hud.style.display = 'none';
				};

				document.getElementById('nd-btn-copy-insp').onclick = function() {
					navigator.clipboard.writeText('http://127.0.0.1:9222').then(function() {
						showFloatingToast('📋 Inspector URL copied: http://127.0.0.1:9222');
					});
				};

				document.getElementById('nd-btn-log-store').onclick = function() {
					console.log('🔬 [ndWhatsApp Store]', window.ndWA.Store);
					showFloatingToast('🔬 Logged window.WA.Store to browser console!');
				};

				document.getElementById('nd-btn-clear-events').onclick = function() {
					window.ndWA.eventLog = [];
					window.ndUpdateLiveEvents();
				};

				// Tab switching
				var tabBtns = hud.querySelectorAll('.nd-tab-btn');
				tabBtns.forEach(function(btn) {
					btn.onclick = function() {
						var targetTab = btn.getAttribute('data-tab');
						tabBtns.forEach(function(b) {
							b.style.background = 'transparent';
							b.style.color = '#8696a0';
							b.style.borderBottomColor = 'transparent';
						});
						btn.style.background = '#202c33';
						btn.style.color = '#00a884';
						btn.style.borderBottomColor = '#00a884';

						hud.querySelectorAll('.nd-tab-content').forEach(function(c) {
							c.style.display = 'none';
						});
						var content = document.getElementById('nd-tab-' + targetTab);
						if (content) content.style.display = 'block';

						if (window.ndUpdateHUD) window.ndUpdateHUD();
					};
				});

				// Runner eval
				var inputEval = document.getElementById('nd-input-eval');
				var btnRun = document.getElementById('nd-btn-run-eval');
				var resultBox = document.getElementById('nd-eval-result');

				function executeSnippet() {
					var expr = (inputEval.value || '').trim();
					if (!expr) return;
					try {
						var out = eval(expr);
						if (out === undefined) {
							resultBox.textContent = 'undefined';
						} else if (typeof out === 'object') {
							resultBox.textContent = JSON.stringify(out, null, 2);
						} else {
							resultBox.textContent = String(out);
						}
						resultBox.style.color = '#00a884';
					} catch (e) {
						resultBox.textContent = 'Error: ' + e.message;
						resultBox.style.color = '#ea4335';
					}
				}
				btnRun.onclick = executeSnippet;
				inputEval.onkeydown = function(e) {
					if (e.key === 'Enter') executeSnippet();
				};
			}

			// Update HUD Data
			window.ndUpdateHUD = function() {
				var badge = document.getElementById('nd-hud-badge-status');
				if (badge) {
					if (window.ndWA.ready) {
						badge.textContent = '● Hooked (' + Object.keys(window.ndWA.rawModules).length + ' modules)';
						badge.style.background = '#0a332c';
						badge.style.color = '#00a884';
					} else {
						badge.textContent = '○ Waiting Webpack...';
						badge.style.background = '#374045';
						badge.style.color = '#ffd279';
					}
				}

				if (window.getProcessMetricsNative) {
					window.getProcessMetricsNative().then(function(m) {
						if (!m) return;
						var p = document.getElementById('nd-stat-pid');
						var a = document.getElementById('nd-stat-alloc');
						var g = document.getElementById('nd-stat-goroutines');
						var s = document.getElementById('nd-stat-sys');
						if (p) p.textContent = m.pid;
						if (a) a.textContent = m.alloc_mb;
						if (g) g.textContent = m.goroutines;
						if (s) s.textContent = m.sys_mb;
					}).catch(function() {});
				}

				// Update Chat list preview
				var chats = window.ndWA.getChats();
				var chatCount = document.getElementById('nd-stat-chats-count');
				if (chatCount) chatCount.textContent = chats.length;

				var chatPreview = document.getElementById('nd-chat-list-preview');
				if (chatPreview && chats.length > 0) {
					var html = '';
					for (var i = 0; i < Math.min(chats.length, 30); i++) {
						var c = chats[i];
						var name = c.name || c.formattedTitle || (c.id ? c.id.user : 'Unknown');
						var unread = c.unreadCount ? (' (' + c.unreadCount + ')') : '';
						html += '<div style="padding:2px 0;border-bottom:1px solid #182229;display:flex;justify-content:space-between;">' +
								'<span style="color:#e9edef;overflow:hidden;text-overflow:ellipsis;white-space:nowrap;max-width:240px;">' + name + unread + '</span>' +
								'<span style="color:#8696a0;font-size:9.5px;">' + (c.id ? c.id.user : '') + '</span>' +
								'</div>';
					}
					chatPreview.innerHTML = html;
				}

				window.ndUpdateLiveEvents();
			};

			window.ndUpdateLiveEvents = function() {
				var container = document.getElementById('nd-event-log-container');
				if (!container) return;
				var logs = window.ndWA.eventLog;
				if (logs.length === 0) {
					container.innerHTML = '<div style="color:#8696a0;text-align:center;padding:10px;">Belum ada pesan masuk sejak aplikasi dibuka.</div>';
					return;
				}
				var html = '';
				for (var i = 0; i < logs.length; i++) {
					var e = logs[i];
					html += '<div style="padding:4px;border-bottom:1px solid #182229;line-height:1.3;">' +
							'<div style="display:flex;justify-content:space-between;color:#8696a0;font-size:9px;">' +
							'  <span>' + e.time + ' [' + e.type + ']</span>' +
							'  <span>' + e.from.slice(0, 16) + '</span>' +
							'</div>' +
							'<div style="color:#00a884;margin-top:2px;">' + (e.body || '<i>(media / non-text)</i>') + '</div>' +
							'</div>';
				}
				container.innerHTML = html;
			};

			// Ensure DevHUD is injected after DOM ready
			document.addEventListener('DOMContentLoaded', injectDevHUD);
			window.addEventListener('load', injectDevHUD);
		})();
	`
}

package main

import "runtime"

const (
	windowWidth  = 1100
	windowHeight = 750
)

func getInitScript(ua string) string {
	clientPlatform := "macOS"
	clientPlatformVersion := "15.0.0"
	clientArch := "arm"
	if runtime.GOOS == "windows" {
		clientPlatform = "Windows"
		clientPlatformVersion = "10.0.0"
		clientArch = "x86"
	} else if runtime.GOOS == "linux" {
		clientPlatform = "Linux"
		clientPlatformVersion = "6.8.0"
		clientArch = "x86"
	}

	return `
		// UserAgent and platform override to Google Chrome
		Object.defineProperty(navigator, 'userAgent', {
			get: () => '` + ua + `'
		});
		Object.defineProperty(navigator, 'appVersion', {
			get: () => '` + ua + `'
		});
		Object.defineProperty(navigator, 'vendor', {
			get: () => 'Google Inc.'
		});

		// Emulate window.chrome
		if (!window.chrome) {
			window.chrome = {
				app: { isInstalled: false },
				runtime: {}
			};
		}

		// Remove Safari-specific markers
		try {
			delete window.safari;
		} catch (e) {}

		// Emulate navigator.userAgentData (User-Agent Client Hints)
		if (!navigator.userAgentData) {
			Object.defineProperty(navigator, 'userAgentData', {
				get: () => ({
					brands: [
						{ brand: 'Not(A:Brand', version: '99' },
						{ brand: 'Google Chrome', version: '133' },
						{ brand: 'Chromium', version: '133' }
					],
					mobile: false,
					platform: '` + clientPlatform + `',
					getHighEntropyValues: function() {
						return Promise.resolve({
							architecture: '` + clientArch + `',
							bitness: '64',
							brands: [
								{ brand: 'Not(A:Brand', version: '99' },
								{ brand: 'Google Chrome', version: '133' },
								{ brand: 'Chromium', version: '133' }
							],
							fullVersionList: [
								{ brand: 'Not(A:Brand', version: '99.0.0.0' },
								{ brand: 'Google Chrome', version: '133.0.0.0' },
								{ brand: 'Chromium', version: '133.0.0.0' }
							],
							mobile: false,
							model: '',
							platform: '` + clientPlatform + `',
							platformVersion: '` + clientPlatformVersion + `',
							uaFullVersion: '133.0.0.0'
						});
					}
				})
			});
		}

		// Keep WKWebView's real PDF capability untouched. Advertising Chrome's
		// PDF plugin makes WhatsApp open a viewer that WKWebView cannot render.

		// Native Notification Polyfill & ServiceWorker Notification Interceptor
		(function() {
			function dispatchNativeNotification(title, options) {
				options = options || {};
				var body = options.body || '';
				if (window.sendNativeNotification) {
					window.sendNativeNotification(title, body);
				}
			}

			window.Notification = function(title, options) {
				options = options || {};
				dispatchNativeNotification(title, options);
				this.title = title;
				this.body = options.body || '';
				this.onclick = null;
				this.onclose = null;
				this.onerror = null;
				this.onshow = null;
			};
			window.Notification.permission = 'granted';
			window.Notification.maxActions = 2;
			window.Notification.requestPermission = function(callback) {
				var p = Promise.resolve('granted');
				if (typeof callback === 'function') {
					callback('granted');
				}
				return p;
			};

			try {
				if (typeof ServiceWorkerRegistration !== 'undefined' && ServiceWorkerRegistration.prototype) {
					ServiceWorkerRegistration.prototype.showNotification = function(title, options) {
						dispatchNativeNotification(title, options);
						return Promise.resolve();
					};
				}
			} catch (e) {}
		})();

		// Robust HTML5 Media Autoplay & Inline Playback Support for Status/Stories and Videos
		(function() {
			if (!window.HTMLMediaElement) return;

			function prepareMedia(el) {
				if (!el || el.__wa_media_ready) return;
				el.__wa_media_ready = true;
				if (el.tagName === 'VIDEO') {
					el.setAttribute('playsinline', '');
					el.setAttribute('webkit-playsinline', '');
					el.setAttribute('x5-playsinline', '');
				}
				if (!el.getAttribute('preload')) {
					el.setAttribute('preload', 'auto');
				}
			}

			var origPlay = HTMLMediaElement.prototype.play;
			HTMLMediaElement.prototype.play = function() {
				var self = this;
				prepareMedia(self);
				var res = origPlay.apply(this, arguments);
				if (res && typeof res.catch === 'function') {
					return res.catch(function(err) {
						// When WebKit blocks unmuted autoplay, mute the media and retry playback
						if (err && (err.name === 'NotAllowedError' || err.name === 'AbortError')) {
							self.muted = true;
							return origPlay.apply(self);
						}
						return Promise.reject(err);
					});
				}
				return res;
			};

			// Automatically prepare all video/audio elements injected into DOM
			if (window.MutationObserver) {
				var mediaObserver = new MutationObserver(function(mutations) {
					for (var m = 0; m < mutations.length; m++) {
						var added = mutations[m].addedNodes;
						for (var n = 0; n < added.length; n++) {
							var node = added[n];
							if (node.nodeType === 1) {
								if (node.tagName === 'VIDEO' || node.tagName === 'AUDIO') {
									prepareMedia(node);
								} else if (node.querySelectorAll) {
									var list = node.querySelectorAll('video, audio');
									for (var l = 0; l < list.length; l++) {
										prepareMedia(list[l]);
									}
								}
							}
						}
					}
				});
				var targetNode = document.documentElement || document.body;
				if (targetNode) {
					mediaObserver.observe(targetNode, { childList: true, subtree: true });
				} else {
					document.addEventListener('DOMContentLoaded', function() {
						mediaObserver.observe(document.body, { childList: true, subtree: true });
					});
				}
			}
		})();

		function isDocumentFileName(name) {
			if (!name) return false;
			var ext = name.toLowerCase();
			return ext.endsWith('.pdf') || ext.endsWith('.doc') || ext.endsWith('.docx') ||
				   ext.endsWith('.xls') || ext.endsWith('.xlsx') || ext.endsWith('.ppt') ||
				   ext.endsWith('.pptx') || ext.endsWith('.txt') || ext.endsWith('.csv') ||
				   ext.endsWith('.rtf');
		}

		// Dismiss WhatsApp Web's internal stuck viewer overlay
		function dismissStuckViewer() {
			var attempts = 0;
			var dismissTimer = setInterval(function() {
				attempts++;
				var viewer = document.querySelector('[data-testid="media-viewer"]');
				if (!viewer || attempts > 15) {
					clearInterval(dismissTimer);
					return;
				}
				var closeSelectors = [
					'button[data-testid="x-viewer"]',
					'[data-testid="x-viewer"]',
					'[data-icon="x-viewer"]',
					'[data-icon="x"]',
					'[data-icon="back"]',
					'button[aria-label*="Close" i]',
					'button[aria-label*="Tutup" i]',
					'[role="button"][aria-label*="Close" i]',
					'[role="button"][aria-label*="Tutup" i]',
					'button[title*="Close" i]',
					'button[title*="Tutup" i]',
					'[data-testid="btn-close"]',
					'[data-testid="media-viewer-close"]'
				];
				for (var i = 0; i < closeSelectors.length; i++) {
					try {
						var el = viewer.querySelector(closeSelectors[i]) || document.querySelector(closeSelectors[i]);
						if (el) {
							var btn = el.closest('button, [role="button"]') || el;
							btn.click();
							clearInterval(dismissTimer);
							return;
						}
					} catch (e) {}
				}
				// One synthetic Escape may close WhatsApp's viewer when its close
				// control is absent. Do not repeatedly send it to document/window:
				// those events also reach our own preview modal.
				if (attempts === 1) {
					var escEvt = new KeyboardEvent('keydown', { key: 'Escape', code: 'Escape', keyCode: 27, which: 27, bubbles: true, cancelable: true });
					try { viewer.dispatchEvent(escEvt); } catch (e) {}
				}
			}, 150);
		}
		window.dismissStuckViewer = dismissStuckViewer;

		// Track clicked document filenames with robust regex matching
		var lastClickedDocName = '';
		var lastDocumentIntentAt = 0;
		function extractDocumentName(el) {
			if (!el || typeof el.closest !== 'function') return '';
			// NEVER extract document names from inside the media viewer, modal dialogs, or top toolbars
			if (el.closest('[data-testid="media-viewer"]') ||
			    el.closest('#wa-doc-modal-overlay') ||
			    el.closest('[role="toolbar"]') ||
			    el.closest('header')) {
				return '';
			}

			// Only search within a chat message container / row / bubble
			var msgContainer = el.closest('[data-testid*="msg-container"], [role="row"], div[data-id], .message-in, .message-out');
			if (!msgContainer) return '';

			var node = el;
			while (node && node !== msgContainer.parentElement && node !== document.body) {
				var title = node.getAttribute && (node.getAttribute('title') || node.getAttribute('aria-label') || '');
				var titleMatch = title && title.match(/([^\n\r<>]{1,180}\.(pdf|docx?|xlsx?|pptx?|txt|csv|rtf))\b/i);
				if (titleMatch && titleMatch[1]) return titleMatch[1].trim();

				// Check text only on leaf-ish nodes to prevent matching unrelated long container text
				if (!node.children || node.children.length < 5) {
					var text = (node.innerText || '').trim();
					if (text.length > 0 && text.length < 250) {
						var textMatch = text.match(/([^\n\r<>]{1,180}\.(pdf|docx?|xlsx?|pptx?|txt|csv|rtf))\b/i);
						if (textMatch && textMatch[1]) return textMatch[1].trim();
					}
				}
				if (node === msgContainer) break;
				node = node.parentElement;
			}
			return '';
		}
		function isRecentPDFIntent() {
			return !!lastClickedDocName && isDocumentFileName(lastClickedDocName) &&
				(Date.now() - lastDocumentIntentAt) < 20000;
		}
		document.addEventListener('click', function(e) {
			var name = extractDocumentName(e.target);
			if (name) {
				lastClickedDocName = name;
				lastDocumentIntentAt = Date.now();
			}
		}, true);

		window.closeDocumentViewerAfterNativePreview = function() {
			var selectors = [
				'button[data-testid="x-viewer"]', '[data-testid="x-viewer"]',
				'[data-icon="x-viewer"]', '[data-icon="x"]', '[data-icon="back"]',
				'button[aria-label*="Close" i]', 'button[aria-label*="Tutup" i]',
				'[role="button"][aria-label*="Close" i]', '[role="button"][aria-label*="Tutup" i]',
				'button[title*="Close" i]', 'button[title*="Tutup" i]'
			].join(',');
			var candidates = document.querySelectorAll(selectors);
			var best = null;
			var bestScore = -1;
			for (var i = 0; i < candidates.length; i++) {
				var raw = candidates[i];
				if (raw.closest && raw.closest('#wa-doc-modal-overlay')) continue;
				var control = (raw.closest && raw.closest('button, [role="button"]')) || raw;
				var rect = control.getBoundingClientRect();
				if (rect.width < 8 || rect.height < 8 || rect.bottom <= 0 || rect.right <= 0 ||
					rect.top >= window.innerHeight || rect.left >= window.innerWidth) continue;
				var style = window.getComputedStyle(control);
				if (style.display === 'none' || style.visibility === 'hidden' || Number(style.opacity) === 0) continue;
				var score = 0;
				if (rect.top < window.innerHeight * 0.3) score += 4;
				if (rect.left > window.innerWidth * 0.7) score += 4;
				if (control.closest && control.closest('[role="dialog"], [data-testid*="viewer"], header, [role="toolbar"]')) score += 5;
				if (score > bestScore) { best = control; bestScore = score; }
			}

			if (best && bestScore >= 8) {
				best.click();
			} else {
				var esc = new KeyboardEvent('keydown', {
					key: 'Escape', code: 'Escape', keyCode: 27, which: 27,
					bubbles: true, cancelable: true
				});
				document.dispatchEvent(esc);
				window.dispatchEvent(esc);
			}
			lastDocumentIntentAt = 0;
			lastClickedDocName = '';
		};

		// Handle explicit user clicks on WhatsApp Web's Media Viewer ✕ close button
		// Guarantees immediate exit to chat view even if internal viewer state is desynced
		document.addEventListener('click', function(e) {
			var target = e.target;
			if (!target || typeof target.closest !== 'function') return;
			var viewer = target.closest('[data-testid="media-viewer"]');
			if (!viewer) return;

			var isCloseBtn = target.closest([
				'button[data-testid="x-viewer"]',
				'[data-testid="x-viewer"]',
				'[data-icon="x-viewer"]',
				'[data-icon="x"]',
				'[data-icon="back"]',
				'button[aria-label*="Close" i]',
				'button[aria-label*="Tutup" i]',
				'button[title*="Close" i]',
				'button[title*="Tutup" i]',
				'[data-testid="btn-close"]'
			].join(','));

			if (isCloseBtn) {
				lastDocumentIntentAt = 0;
				lastClickedDocName = '';
				setTimeout(function() {
					var activeViewer = document.querySelector('[data-testid="media-viewer"]');
					if (activeViewer) {
						var escEvt = new KeyboardEvent('keydown', {
							key: 'Escape',
							code: 'Escape',
							keyCode: 27,
							which: 27,
							bubbles: true,
							cancelable: true
						});
						document.dispatchEvent(escEvt);
						window.dispatchEvent(escEvt);
					}
				}, 60);
			}
		}, false);

		// Intercept external link clicks to open in default browser
		document.addEventListener('click', function(e) {
			var target = e.target;
			while (target && target !== document.body && target.tagName !== 'A') {
				target = target.parentElement;
			}
			if (target && target.tagName === 'A' && target.href) {
				try {
					var url = new URL(target.href);
					if (!url.hostname.endsWith('whatsapp.com') && !url.hostname.endsWith('whatsapp.net') && (url.protocol === 'http:' || url.protocol === 'https:')) {
						e.preventDefault();
						e.stopPropagation();
						if (window.openExternalLink) {
							window.openExternalLink(target.href);
						}
					}
				} catch(err) {}
			}
		}, true);

		// Helper: Decode base64 dataURI to Uint8Array
		function base64ToUint8Array(dataUri) {
			try {
				var base64 = dataUri.indexOf(';base64,') !== -1 ? dataUri.split(';base64,')[1] : dataUri;
				var binary = atob(base64);
				var bytes = new Uint8Array(binary.length);
				for (var i = 0; i < binary.length; i++) bytes[i] = binary.charCodeAt(i);
				return bytes;
			} catch (e) {
				return null;
			}
		}

		// Helper: Read a specific file from ZIP payload (e.g. word/document.xml, xl/worksheets/sheet1.xml)
		async function readZipEntryText(uint8Array, targetPath) {
			if (!uint8Array || uint8Array.length < 30) return null;
			try {
				var view = new DataView(uint8Array.buffer, uint8Array.byteOffset, uint8Array.byteLength);
				var offset = 0;
				while (offset < uint8Array.length - 30) {
					if (view.getUint32(offset, true) === 0x04034b50) {
						var compMethod = view.getUint16(offset + 8, true);
						var compSize = view.getUint32(offset + 18, true);
						var nameLen = view.getUint16(offset + 26, true);
						var extraLen = view.getUint16(offset + 28, true);
						var nameBytes = uint8Array.subarray(offset + 30, offset + 30 + nameLen);
						var name = new TextDecoder().decode(nameBytes);
						var dataStart = offset + 30 + nameLen + extraLen;
						var dataEnd = dataStart + compSize;

						if (name.toLowerCase() === targetPath.toLowerCase()) {
							var compressedData = uint8Array.subarray(dataStart, dataEnd);
							if (compMethod === 0) {
								return new TextDecoder().decode(compressedData);
							} else if (compMethod === 8 && typeof DecompressionStream !== 'undefined') {
								var ds = new DecompressionStream('deflate-raw');
								var writer = ds.writable.getWriter();
								writer.write(compressedData);
								writer.close();
								var response = new Response(ds.readable);
								return await response.text();
							}
						}
						offset = dataEnd;
					} else {
						offset++;
					}
				}
			} catch (e) {
				console.warn('Zip read error:', e);
			}
			return null;
		}

		function parseDocxToHtml(xmlStr) {
			if (!xmlStr) return '';
			var pMatches = xmlStr.match(/<w:p\b[\s\S]*?<\/w:p>/g) || [];
			var html = [];
			for (var i = 0; i < pMatches.length; i++) {
				var pStr = pMatches[i];
				var isH1 = /<w:pStyle\b[^>]*w:val="Heading1"/i.test(pStr);
				var isH2 = /<w:pStyle\b[^>]*w:val="Heading2"/i.test(pStr);
				var isH3 = /<w:pStyle\b[^>]*w:val="Heading[3-6]"/i.test(pStr);
				var rMatches = pStr.match(/<w:r\b[\s\S]*?<\/w:r>/g) || [];
				var pText = '';
				for (var j = 0; j < rMatches.length; j++) {
					var rStr = rMatches[j];
					var isBold = /<w:b\b/.test(rStr);
					var isItalic = /<w:i\b/.test(rStr);
					var tMatches = rStr.match(/<w:t\b[^>]*>([\s\S]*?)<\/w:t>/g) || [];
					for (var k = 0; k < tMatches.length; k++) {
						var tVal = tMatches[k].replace(/<w:t\b[^>]*>|<\/w:t>/g, '');
						tVal = tVal.replace(/&/g, '&amp;').replace(/</g, '&lt;').replace(/>/g, '&gt;');
						if (isBold) tVal = '<strong>' + tVal + '</strong>';
						if (isItalic) tVal = '<em>' + tVal + '</em>';
						pText += tVal;
					}
				}
				if (pText.trim()) {
					if (isH1) html.push('<h2 style="color:#111b21;margin:18px 0 8px;font-size:18px;font-weight:700;">' + pText + '</h2>');
					else if (isH2) html.push('<h3 style="color:#111b21;margin:14px 0 6px;font-size:16px;font-weight:600;">' + pText + '</h3>');
					else if (isH3) html.push('<h4 style="color:#111b21;margin:12px 0 4px;font-size:14px;font-weight:600;">' + pText + '</h4>');
					else html.push('<p style="color:#222e35;margin:8px 0;line-height:1.65;font-size:13.5px;">' + pText + '</p>');
				}
			}
			return html.join('');
		}

		function parseXlsxToHtml(sheetXml, stringsXml) {
			if (!sheetXml) return '';
			var sharedStrings = [];
			if (stringsXml) {
				var siMatches = stringsXml.match(/<si\b[\s\S]*?<\/si>/g) || [];
				for (var s = 0; s < siMatches.length; s++) {
					var tMatches = siMatches[s].match(/<t\b[^>]*>([\s\S]*?)<\/t>/g) || [];
					var strVal = '';
					for (var tm = 0; tm < tMatches.length; tm++) {
						strVal += tMatches[tm].replace(/<t\b[^>]*>|<\/t>/g, '');
					}
					sharedStrings.push(strVal);
				}
			}

			var rowMatches = sheetXml.match(/<row\b[\s\S]*?<\/row>/g) || [];
			if (!rowMatches.length) return '<div style="padding:20px;color:#8696a0;">Spreadsheet is empty.</div>';

			var colMap = {};
			var parsedRows = [];

			for (var r = 0; r < Math.min(rowMatches.length, 300); r++) {
				var rStr = rowMatches[r];
				var rowObj = {};
				var cellMatches = rStr.match(/<c\b[\s\S]*?<\/c>|<c\b[^>]*\/>/g) || [];
				for (var c = 0; c < cellMatches.length; c++) {
					var cStr = cellMatches[c];
					var refMatch = cStr.match(/r="([A-Z]+)(\d+)"/);
					if (!refMatch) continue;
					var colLetter = refMatch[1];
					colMap[colLetter] = true;
					var isShared = cStr.indexOf('t="s"') !== -1;
					var vMatch = cStr.match(/<v>([\s\S]*?)<\/v>/);
					var val = vMatch ? vMatch[1] : '';
					if (isShared && sharedStrings[parseInt(val, 10)] !== undefined) {
						val = sharedStrings[parseInt(val, 10)];
					}
					rowObj[colLetter] = val;
				}
				parsedRows.push(rowObj);
			}

			var cols = Object.keys(colMap).sort(function(a, b) {
				if (a.length !== b.length) return a.length - b.length;
				return a.localeCompare(b);
			});
			if (!cols.length) cols = ['A', 'B', 'C', 'D', 'E'];

			var tableHtml = '<div style="width:100%;height:100%;overflow:auto;background:#111b21;">' +
				'<table style="width:100%;border-collapse:collapse;font-family:system-ui,-apple-system,sans-serif;font-size:12px;color:#e9edef;table-layout:auto;">' +
				'<thead><tr style="background:#202c33;position:sticky;top:0;z-index:2;box-shadow:0 1px 0 #2a3942;">' +
				'<th style="width:40px;padding:6px 8px;border:1px solid #2a3942;color:#8696a0;text-align:center;font-weight:600;font-size:11px;">#</th>';

			for (var ci = 0; ci < cols.length; ci++) {
				tableHtml += '<th style="padding:6px 10px;border:1px solid #2a3942;color:#00a884;text-align:center;font-weight:600;min-width:90px;">' + cols[ci] + '</th>';
			}
			tableHtml += '</tr></thead><tbody>';

			for (var ri = 0; ri < parsedRows.length; ri++) {
				var row = parsedRows[ri];
				var bg = ri % 2 === 0 ? '#111b21' : '#182229';
				tableHtml += '<tr style="background:' + bg + ';">' +
					'<td style="padding:5px 8px;border:1px solid #2a3942;color:#8696a0;text-align:center;font-weight:600;font-size:10.5px;">' + (ri + 1) + '</td>';
				for (var cj = 0; cj < cols.length; cj++) {
					var cellVal = row[cols[cj]] || '';
					var escVal = cellVal.replace(/&/g, '&amp;').replace(/</g, '&lt;').replace(/>/g, '&gt;');
					tableHtml += '<td style="padding:6px 10px;border:1px solid #2a3942;white-space:nowrap;max-width:320px;overflow:hidden;text-overflow:ellipsis;">' + escVal + '</td>';
				}
				tableHtml += '</tr>';
			}

			tableHtml += '</tbody></table></div>';
			return tableHtml;
		}

		function parseCsvToHtml(csvText) {
			if (!csvText) return '';
			var lines = csvText.split(/\r?\n/).filter(function(l) { return l.trim().length > 0; });
			if (!lines.length) return '<div style="padding:20px;color:#8696a0;">CSV file is empty.</div>';

			var delimiter = lines[0].indexOf(';') !== -1 ? ';' : ',';
			var tableHtml = '<div style="width:100%;height:100%;overflow:auto;background:#111b21;">' +
				'<table style="width:100%;border-collapse:collapse;font-family:system-ui,-apple-system,sans-serif;font-size:12px;color:#e9edef;">' +
				'<thead><tr style="background:#202c33;position:sticky;top:0;z-index:2;box-shadow:0 1px 0 #2a3942;">' +
				'<th style="width:40px;padding:6px 8px;border:1px solid #2a3942;color:#8696a0;text-align:center;font-size:11px;">#</th>';

			var headerCols = lines[0].split(delimiter);
			for (var h = 0; h < headerCols.length; h++) {
				var hName = headerCols[h].replace(/^["']|["']$/g, '').trim();
				tableHtml += '<th style="padding:6px 10px;border:1px solid #2a3942;color:#00a884;text-align:left;font-weight:600;min-width:100px;">' + hName + '</th>';
			}
			tableHtml += '</tr></thead><tbody>';

			for (var i = 1; i < Math.min(lines.length, 300); i++) {
				var cols = lines[i].split(delimiter);
				var bg = i % 2 === 0 ? '#111b21' : '#182229';
				tableHtml += '<tr style="background:' + bg + ';"><td style="padding:5px 8px;border:1px solid #2a3942;color:#8696a0;text-align:center;font-size:10.5px;">' + i + '</td>';
				for (var j = 0; j < headerCols.length; j++) {
					var val = (cols[j] || '').replace(/^["']|["']$/g, '').trim();
					val = val.replace(/&/g, '&amp;').replace(/</g, '&lt;').replace(/>/g, '&gt;');
					tableHtml += '<td style="padding:6px 10px;border:1px solid #2a3942;white-space:nowrap;">' + val + '</td>';
				}
				tableHtml += '</tr>';
			}
			tableHtml += '</tbody></table></div>';
			return tableHtml;
		}

		// In-App Document Preview Modal Overlay (PDF, Excel, Word, Text)
		function showInAppDocModal(filename, blobUrl, savedPath, dataUri, ownedBlobUrl) {
			var existing = document.getElementById('wa-doc-modal-overlay');
			if (existing && existing.parentNode) existing.parentNode.removeChild(existing);

			// Immediately dismiss WhatsApp Web's stuck background viewer
			if (window.dismissStuckViewer) window.dismissStuckViewer();

			var ext = (filename && filename.indexOf('.') !== -1 ? filename.split('.').pop() : '').toLowerCase();
			var isPdf = ext === 'pdf';
			var isExcel = ext === 'xlsx' || ext === 'xls' || ext === 'csv';
			var isWord = ext === 'docx' || ext === 'doc' || ext === 'rtf' || ext === 'txt';

			// WKWebView has no reliable built-in renderer for PDF blob URLs.
			// On macOS, render the already-saved file with PDFKit instead.
			if (isPdf && savedPath && window.showPDFPreviewNative) {
				window.showPDFPreviewNative(savedPath);
				if (ownedBlobUrl) {
					try { URL.revokeObjectURL(ownedBlobUrl); } catch (e) {}
				}
				return;
			}

			var docIcon = '📄';
			var openBtnText = '📂 Open in System App';
			var docTypeLabel = 'PDF Document';
			if (isExcel) {
				docIcon = '📊';
				openBtnText = '📊 Open in Excel / Numbers';
				docTypeLabel = 'Excel Spreadsheet';
			} else if (isWord) {
				docIcon = '📝';
				openBtnText = '📝 Open in Word / Pages';
				docTypeLabel = 'Word Document';
			}

			var overlay = document.createElement('div');
			overlay.id = 'wa-doc-modal-overlay';
			overlay.style.cssText = 'position:fixed;top:0;left:0;width:100%;height:100%;background:rgba(0,0,0,0.85);backdrop-filter:blur(10px);-webkit-backdrop-filter:blur(10px);z-index:99999999;display:flex;flex-direction:column;align-items:center;justify-content:center;padding:16px;box-sizing:border-box;animation:waFadeIn 0.2s ease;';

			var modal = document.createElement('div');
			modal.style.cssText = 'width:94%;max-width:1020px;height:92%;background:#111b21;border:1px solid rgba(255,255,255,0.14);border-radius:12px;display:flex;flex-direction:column;overflow:hidden;box-shadow:0 24px 60px rgba(0,0,0,0.85);';

			// Header
			var header = document.createElement('div');
			header.style.cssText = 'display:flex;align-items:center;justify-content:space-between;padding:10px 16px;border-bottom:1px solid rgba(255,255,255,0.08);background:#202c33;flex-shrink:0;';
			header.innerHTML = '' +
				'<div style="display:flex;align-items:center;gap:10px;min-width:0;">' +
				'  <span style="font-size:22px;">' + docIcon + '</span>' +
				'  <div style="min-width:0;">' +
				'    <strong style="font-size:13.5px;color:#e9edef;white-space:nowrap;overflow:hidden;text-overflow:ellipsis;display:block;max-width:420px;" title="' + filename + '">' + filename + '</strong>' +
				'    <span style="font-size:11px;color:#8696a0;">' + docTypeLabel + ' · Direct Preview</span>' +
				'  </div>' +
				'</div>' +
				'<div style="display:flex;align-items:center;gap:8px;">' +
				'  <button id="wa-btn-open-preview" style="background:#00a884;color:#111b21;border:none;padding:6px 14px;border-radius:6px;font-size:12px;font-weight:600;cursor:pointer;display:flex;align-items:center;gap:4px;box-shadow:0 2px 6px rgba(0,168,132,0.3);">' +
				'    ' + openBtnText +
				'  </button>' +
				'  <button id="wa-btn-save-doc" style="background:#2a3942;color:#e9edef;border:1px solid rgba(255,255,255,0.1);padding:6px 14px;border-radius:6px;font-size:12px;font-weight:500;cursor:pointer;">' +
				'    💾 Download' +
				'  </button>' +
				'  <button id="wa-btn-close-doc" style="background:transparent;border:none;color:#8696a0;cursor:pointer;font-size:20px;padding:4px 8px;border-radius:6px;line-height:1;">✕</button>' +
				'</div>';
			modal.appendChild(header);

			// Body Container
			var body = document.createElement('div');
			body.style.cssText = 'flex:1;width:100%;height:100%;position:relative;background:#0c1317;overflow:hidden;display:flex;flex-direction:column;align-items:center;justify-content:center;';
			modal.appendChild(body);

			function triggerOpenSystem() {
				if (savedPath && window.showPDFPreviewNative && isPdf) {
					window.showPDFPreviewNative(savedPath);
				} else if (savedPath && window.openFileNative) {
					window.openFileNative(savedPath);
				} else if (window.previewDocumentNative) {
					window.previewDocumentNative(filename, dataUri || blobUrl);
				}
			}

			function renderCardFallback(hint) {
				body.innerHTML = '' +
					'<div style="display:flex;flex-direction:column;align-items:center;justify-content:center;padding:40px;text-align:center;">' +
					'  <div style="font-size:64px;margin-bottom:16px;">' + docIcon + '</div>' +
					'  <h2 style="color:#e9edef;font-size:17px;font-weight:600;margin:0 0 8px;">' + filename + '</h2>' +
					'  <p style="color:#8696a0;font-size:12.5px;max-width:420px;line-height:1.5;margin:0 0 24px;">' +
					(hint || ('The ' + docTypeLabel + ' file is saved on your computer. Click the button below to open it.')) +
					'  </p>' +
					'  <button id="wa-btn-card-launch" style="background:#00a884;color:#111b21;border:none;padding:10px 24px;border-radius:8px;font-size:13.5px;font-weight:600;cursor:pointer;display:flex;align-items:center;gap:6px;box-shadow:0 4px 12px rgba(0,168,132,0.3);">' +
					openBtnText +
					'  </button>' +
					'</div>';
				var cardBtn = document.getElementById('wa-btn-card-launch');
				if (cardBtn) cardBtn.onclick = triggerOpenSystem;
			}

			// Render content according to file type
			if (isPdf) {
				var pdfSrc = ownedBlobUrl || blobUrl || '';
				if ((!pdfSrc || pdfSrc.indexOf('blob:') !== 0) && dataUri && dataUri.indexOf('data:application/pdf') === 0) {
					try {
						var rawB64 = dataUri.split(',')[1] || '';
						var binStr = atob(rawB64);
						var len = binStr.length;
						var u8 = new Uint8Array(len);
						for (var bi = 0; bi < len; bi++) {
							u8[bi] = binStr.charCodeAt(bi);
						}
						var pBlob = new Blob([u8], { type: 'application/pdf' });
						pdfSrc = URL.createObjectURL(pBlob);
						if (!ownedBlobUrl) ownedBlobUrl = pdfSrc;
					} catch (e) {
						pdfSrc = dataUri;
					}
				}
				if (!pdfSrc) pdfSrc = dataUri || '';

				if (pdfSrc) {
					var obj = document.createElement('object');
					obj.data = pdfSrc;
					obj.type = 'application/pdf';
					obj.style.cssText = 'width:100%;height:100%;border:none;flex:1;';

					var ifr = document.createElement('iframe');
					ifr.src = pdfSrc;
					ifr.style.cssText = 'width:100%;height:100%;border:none;background:#ffffff;';
					ifr.title = filename;

					obj.appendChild(ifr);
					body.appendChild(obj);
				} else {
					renderCardFallback();
				}
			} else if (ext === 'csv') {
				try {
					var rawBase64 = (dataUri || '').split(',')[1] || '';
					var csvText = atob(rawBase64);
					body.innerHTML = parseCsvToHtml(csvText);
				} catch (e) {
					renderCardFallback();
				}
			} else if (ext === 'xlsx') {
				body.innerHTML = '<div style="color:#8696a0;font-size:13px;display:flex;align-items:center;gap:8px;">⏳ Loading Excel preview...</div>';
				var uint8 = base64ToUint8Array(dataUri || '');
				if (uint8) {
					Promise.all([
						readZipEntryText(uint8, 'xl/worksheets/sheet1.xml'),
						readZipEntryText(uint8, 'xl/sharedStrings.xml')
					]).then(function(res) {
						var sheetXml = res[0];
						var stringsXml = res[1];
						if (sheetXml) {
							body.innerHTML = parseXlsxToHtml(sheetXml, stringsXml);
						} else {
							renderCardFallback();
						}
					}).catch(function() {
						renderCardFallback();
					});
				} else {
					renderCardFallback();
				}
			} else if (ext === 'docx') {
				body.innerHTML = '<div style="color:#8696a0;font-size:13px;display:flex;align-items:center;gap:8px;">⏳ Loading Word preview...</div>';
				var uint8Doc = base64ToUint8Array(dataUri || '');
				if (uint8Doc) {
					readZipEntryText(uint8Doc, 'word/document.xml').then(function(docXml) {
						if (docXml) {
							var docHtml = parseDocxToHtml(docXml);
							body.innerHTML = '' +
								'<div style="width:100%;height:100%;overflow-y:auto;padding:24px 16px;box-sizing:border-box;display:flex;justify-content:center;background:#0c1317;">' +
								'  <div style="width:100%;max-width:760px;background:#ffffff;border-radius:6px;box-shadow:0 4px 20px rgba(0,0,0,0.5);padding:40px 48px;box-sizing:border-box;min-height:90%;">' +
								docHtml +
								'  </div>' +
								'</div>';
						} else {
							renderCardFallback();
						}
					}).catch(function() {
						renderCardFallback();
					});
				} else {
					renderCardFallback();
				}
			} else if (ext === 'txt') {
				try {
					var rawTxt = atob((dataUri || '').split(',')[1] || '');
					body.innerHTML = '<div style="width:100%;height:100%;overflow:auto;padding:24px;box-sizing:border-box;background:#111b21;color:#e9edef;font-family:monospace;font-size:13px;line-height:1.6;white-space:pre-wrap;">' +
						rawTxt.replace(/&/g, '&amp;').replace(/</g, '&lt;').replace(/>/g, '&gt;') +
						'</div>';
				} catch (e) {
					renderCardFallback();
				}
			} else {
				renderCardFallback();
			}

			overlay.appendChild(modal);
			document.body.appendChild(overlay);

			function closeDocModal() {
				if (overlay.parentNode) overlay.parentNode.removeChild(overlay);
				if (ownedBlobUrl) {
					try { URL.revokeObjectURL(ownedBlobUrl); } catch (e) {}
					ownedBlobUrl = '';
				}
				dataUri = '';
				if (window.dismissStuckViewer) window.dismissStuckViewer();
			}

			document.getElementById('wa-btn-close-doc').onclick = closeDocModal;
			overlay.onclick = function(e) {
				if (e.target === overlay) closeDocModal();
			};

			document.getElementById('wa-btn-open-preview').onclick = triggerOpenSystem;

			document.getElementById('wa-btn-save-doc').onclick = function() {
				if (dataUri && window.saveDownloadedFileNative) {
					window.saveDownloadedFileNative(filename, dataUri).then(function(p) {
						if (p) showFloatingToast('💾 Saved: ' + filename);
					});
				} else if (savedPath) {
					showFloatingToast('💾 File is saved at: ' + savedPath);
				}
			};

			var onEsc = function(e) {
				if (e.key === 'Escape' && e.isTrusted) {
					closeDocModal();
					window.removeEventListener('keydown', onEsc);
				}
			};
			window.addEventListener('keydown', onEsc);
		}
		window.showInAppDocModal = showInAppDocModal;

		// Intercept URL.createObjectURL to catch decrypted PDF/document blobs directly
		var origCreateObjectURL = URL.createObjectURL;
		URL.createObjectURL = function(blob) {
			var url = origCreateObjectURL.apply(this, arguments);
			try {
				var bType = (blob && blob.type) ? blob.type.toLowerCase() : '';
				var isDocBlob = bType.indexOf('pdf') >= 0 || bType.indexOf('officedocument') >= 0 ||
					bType.indexOf('msword') >= 0 || bType.indexOf('ms-excel') >= 0 ||
					bType.indexOf('spreadsheet') >= 0 || bType.indexOf('wordprocessing') >= 0 ||
					bType === 'text/csv' || bType === 'text/plain' ||
					(blob && (blob.type === 'application/octet-stream' || bType === '') && isRecentPDFIntent());

				if (blob && isDocBlob) {
					var name = lastClickedDocName || 'document';
					if (!name.includes('.')) {
						if (bType.indexOf('pdf') >= 0) name += '.pdf';
						else if (bType.indexOf('sheet') >= 0 || bType.indexOf('excel') >= 0) name += '.xlsx';
						else if (bType.indexOf('word') >= 0) name += '.docx';
						else name += '.pdf';
					}
					var isPdf = name.toLowerCase().endsWith('.pdf');
					var previewBlob = isPdf ? blob.slice(0, blob.size, 'application/pdf') : blob;
					var ownedBlobUrl = isPdf ? origCreateObjectURL(previewBlob) : '';
					var reader = new FileReader();
					reader.onloadend = function() {
						var base64data = reader.result;
						if (window.saveDownloadedFileNative) {
							window.saveDownloadedFileNative(name, base64data).then(function(savedPath) {
								showInAppDocModal(name, ownedBlobUrl, savedPath, base64data, ownedBlobUrl);
								dismissStuckViewer();
								showFloatingToast('📄 Document preview: ' + name);
							});
						} else {
							showInAppDocModal(name, ownedBlobUrl, '', base64data, ownedBlobUrl);
							dismissStuckViewer();
						}
					};
					reader.readAsDataURL(blob);
				}
			} catch (e) {}
			return url;
		};

		function handleBlobDocumentPreview(blobUrl) {
			var name = lastClickedDocName || 'document.pdf';
			fetch(blobUrl)
				.then(function(res) { return res.blob(); })
				.then(function(blob) {
					var isPdf = name.toLowerCase().endsWith('.pdf');
					var previewBlob = isPdf ? blob.slice(0, blob.size, 'application/pdf') : blob;
					var ownedBlobUrl = isPdf ? origCreateObjectURL(previewBlob) : '';
					var reader = new FileReader();
					reader.onloadend = function() {
						var base64data = reader.result;
						if (window.saveDownloadedFileNative) {
							window.saveDownloadedFileNative(name, base64data).then(function(savedPath) {
								showInAppDocModal(name, ownedBlobUrl, savedPath, base64data, ownedBlobUrl);
								dismissStuckViewer();
								showFloatingToast('📄 Document preview: ' + name);
							});
						} else {
							showInAppDocModal(name, ownedBlobUrl, '', base64data, ownedBlobUrl);
							dismissStuckViewer();
						}
					};
					reader.readAsDataURL(blob);
				})
				.catch(function(err) {
					console.error('Error handling blob preview:', err);
				});
		}

		// Intercept window.open for Blob URLs (PDF/Document previews) and external URLs
		var origWindowOpen = window.open;
		window.open = function(url, target, features) {
			if (url && typeof url === 'string') {
				if (url.indexOf('blob:') === 0) {
					handleBlobDocumentPreview(url);
					return null;
				}
				try {
					var parsed = new URL(url, window.location.href);
					if (!parsed.hostname.endsWith('whatsapp.com') && !parsed.hostname.endsWith('whatsapp.net') && (parsed.protocol === 'http:' || parsed.protocol === 'https:')) {
						if (window.openExternalLink) {
							window.openExternalLink(parsed.href);
							return null;
						}
					}
				} catch(err) {}
			}
			return origWindowOpen.apply(this, arguments);
		};

		// Zoom Keyboard Shortcuts (Cmd + / Cmd - / Cmd 0)
		(function() {
			var currentZoom = 1.0;
			window.addEventListener('keydown', function(e) {
				if (e.metaKey || e.ctrlKey) {
					if (e.key === '=' || e.key === '+') {
						e.preventDefault();
						currentZoom = Math.min(currentZoom + 0.1, 2.0);
						document.body.style.zoom = currentZoom;
					} else if (e.key === '-') {
						e.preventDefault();
						currentZoom = Math.max(currentZoom - 0.1, 0.6);
						document.body.style.zoom = currentZoom;
					} else if (e.key === '0') {
						e.preventDefault();
						currentZoom = 1.0;
						document.body.style.zoom = currentZoom;
					}
				}
			});
		})();

		// Dock Badge Unread Count Synchronizer
		(function() {
			var lastBadge = null;
			function syncBadge() {
				var title = document.title || '';
				var match = title.match(/\(([^)]+)\)/);
				var badge = match ? match[1] : '';
				if (badge !== lastBadge) {
					lastBadge = badge;
					if (window.updateDockBadge) {
						window.updateDockBadge(badge);
					}
				}
			}
			var titleEl = document.querySelector('title');
			if (titleEl) {
				new MutationObserver(syncBadge).observe(titleEl, { childList: true, characterData: true, subtree: true });
			} else {
				setInterval(syncBadge, 3000);
			}
		})();

		// Memory Optimization: Idle Garbage Collection
		(function() {
			var releaseTimer = null;
			document.addEventListener('visibilitychange', function() {
				clearTimeout(releaseTimer);
				if (!document.hidden) return;
				lastClickedDocName = '';
				lastDocumentIntentAt = 0;
				releaseTimer = setTimeout(function() {
					if (typeof window.gc === 'function') window.gc();
					if (window.releaseMemoryNative) window.releaseMemoryNative();
				}, 1500);
			});
		})();

		// Debounced window resize persistence
		(function() {
			var resizeTimer = null;
			window.addEventListener('resize', function() {
				clearTimeout(resizeTimer);
				resizeTimer = setTimeout(function() {
					if (window.saveWindowStateNative) {
						var w = window.outerWidth || window.innerWidth;
						var h = window.outerHeight || window.innerHeight;
						if (w && h) {
							window.saveWindowStateNative(Math.round(w), Math.round(h));
						}
					}
				}, 500);
			});
		})();

		// Floating HUD Toast for User Feedback
		function showFloatingToast(msg) {
			var toast = document.getElementById('wa-hud-toast');
			if (!toast) {
				toast = document.createElement('div');
				toast.id = 'wa-hud-toast';
				toast.style.cssText = 'position:fixed;top:16px;left:50%;transform:translateX(-50%);background:rgba(32,44,51,0.94);backdrop-filter:blur(10px);color:#00a884;border:1px solid rgba(0,168,132,0.4);border-radius:20px;padding:8px 20px;font-size:12.5px;font-weight:600;z-index:9999999;box-shadow:0 8px 24px rgba(0,0,0,0.6);pointer-events:none;transition:all 0.22s cubic-bezier(0.16,1,0.3,1);opacity:0;';
				var parent = document.body || document.documentElement;
				if (parent) parent.appendChild(toast);
			}
			if (!toast) return;
			toast.textContent = msg;
			toast.style.opacity = '1';
			toast.style.transform = 'translateX(-50%) translateY(4px)';
			clearTimeout(toast._timer);
			toast._timer = setTimeout(function() {
				toast.style.opacity = '0';
				toast.style.transform = 'translateX(-50%) translateY(0)';
			}, 2500);
		}

		// Privacy Mode Toggle (Cmd + Shift + P)
		(function() {
			var isPrivacyActive = false;
			var styleEl = document.createElement('style');
			styleEl.id = 'whatsapp-privacy-style';
			styleEl.textContent = '.privacy-mode #main .copyable-text, .privacy-mode #main img, .privacy-mode #main video, .privacy-mode #pane-side span[title] { filter: blur(8px) !important; transition: filter 0.15s ease-in-out; } .privacy-mode #main .copyable-text:hover, .privacy-mode #main img:hover, .privacy-mode #main video:hover, .privacy-mode #pane-side span[title]:hover { filter: none !important; }';

			window.togglePrivacyMode = function() {
				isPrivacyActive = !isPrivacyActive;
				if (isPrivacyActive) {
					if (!document.getElementById('whatsapp-privacy-style')) {
						document.head.appendChild(styleEl);
					}
					document.body.classList.add('privacy-mode');
					showFloatingToast('🔒 Privacy Mode: Enabled');
				} else {
					document.body.classList.remove('privacy-mode');
					showFloatingToast('🔓 Privacy Mode: Disabled');
				}
				return isPrivacyActive;
			};
			window.isPrivacyModeActive = function() {
				return isPrivacyActive;
			};

			window.addEventListener('keydown', function(e) {
				if ((e.metaKey || e.ctrlKey) && e.shiftKey && (e.key === 'p' || e.key === 'P')) {
					e.preventDefault();
					window.togglePrivacyMode();
				}
			});
		})();

		// Always on Top Toggle (Cmd/Ctrl + Shift + T)
		(function() {
			var isPinnedState = false;
			window.toggleAlwaysOnTop = function() {
				if (window.toggleAlwaysOnTopNative) {
					return window.toggleAlwaysOnTopNative().then(function(isPinned) {
						isPinnedState = isPinned;
						showFloatingToast(isPinned ? '📌 Always on Top: Enabled' : '📌 Always on Top: Disabled');
						return isPinned;
					});
				}
				return Promise.resolve(false);
			};
			window.isAlwaysOnTopActive = function() {
				return isPinnedState;
			};

			window.addEventListener('keydown', function(e) {
				if ((e.metaKey || e.ctrlKey) && e.shiftKey && (e.key === 't' || e.key === 'T')) {
					e.preventDefault();
					window.toggleAlwaysOnTop();
				}
			});
		})();

		// Reload and Refresh Functions (Cmd/Ctrl + R, Cmd/Ctrl + Shift + R, F5)
		window.reloadWhatsApp = function() {
			showFloatingToast('🔄 Reloading conversation...');
			setTimeout(function() { window.location.reload(); }, 200);
		};
		window.hardRefreshWhatsApp = function() {
			showFloatingToast('⚡ Hard refresh (clearing cache)...');
			try {
				if (window.caches && caches.keys) {
					caches.keys().then(function(names) {
						names.forEach(function(name) { caches.delete(name); });
					});
				}
			} catch (e) {}
			setTimeout(function() {
				window.location.href = window.location.origin + window.location.pathname + '?_t=' + Date.now();
			}, 200);
		};

		window.addEventListener('keydown', function(e) {
			if (e.key === 'F5' || ((e.metaKey || e.ctrlKey) && (e.key === 'r' || e.key === 'R') && !e.shiftKey && !e.altKey)) {
				e.preventDefault();
				window.reloadWhatsApp();
			} else if ((e.metaKey || e.ctrlKey) && e.shiftKey && (e.key === 'r' || e.key === 'R')) {
				e.preventDefault();
				window.hardRefreshWhatsApp();
			}
		});

		// Audio Mute Toggle (Cmd/Ctrl + Shift + M)
		(function() {
			var isMuted = false;
			window.toggleMuteAudio = function() {
				isMuted = !isMuted;
				document.querySelectorAll('audio, video').forEach(function(el) {
					el.muted = isMuted;
				});
				showFloatingToast(isMuted ? '🔇 Notification Audio: Muted' : '🔊 Notification Audio: Unmuted');
				return isMuted;
			};
			window.isAudioMuted = function() {
				return isMuted;
			};

			window.addEventListener('keydown', function(e) {
				if ((e.metaKey || e.ctrlKey) && e.shiftKey && (e.key === 'm' || e.key === 'M')) {
					e.preventDefault();
					window.toggleMuteAudio();
				}
			});
			document.addEventListener('play', function(e) {
				if (isMuted && e.target && (e.target.tagName === 'AUDIO' || e.target.tagName === 'VIDEO')) {
					e.target.muted = true;
				}
			}, true);
		})();

		// Auto-Start at Login Toggle (Cmd/Ctrl + Shift + S)
		(function() {
			var isAutoStartState = false;
			window.toggleAutoStart = function() {
				if (window.toggleAutoStartNative) {
					return window.toggleAutoStartNative().then(function(isEnabled) {
						isAutoStartState = isEnabled;
						showFloatingToast(isEnabled ? '🚀 Launch on Boot: Enabled' : '🚀 Launch on Boot: Disabled');
						return isEnabled;
					});
				}
				return Promise.resolve(false);
			};
			window.isAutoStartActive = function() {
				return isAutoStartState;
			};

			window.addEventListener('keydown', function(e) {
				if ((e.metaKey || e.ctrlKey) && e.shiftKey && (e.key === 's' || e.key === 'S')) {
					e.preventDefault();
					window.toggleAutoStart();
				}
			});
		})();

		// Dynamic Responsive Desktop Layout & Resizable Chat List
		(function() {
			var respStyle = document.createElement('style');
			respStyle.id = 'whatsapp-desktop-responsive';
			respStyle.textContent = '' +
				'html, body, #app { width: 100% !important; height: 100% !important; min-width: 0 !important; overflow: hidden !important; -webkit-font-smoothing: antialiased; }' +
				'#app > div, #app .two { width: 100% !important; height: 100% !important; min-width: 0 !important; max-width: 100% !important; top: 0 !important; margin: 0 !important; border-radius: 0 !important; }' +
				'[data-testid="status-v3"] { min-width: 0 !important; width: 100% !important; height: 100% !important; }' +
				'@media screen and (min-width: 641px) {' +
				'  #wa-side-column, .two > div.x47corl > div:first-child, .two > div.x1o0tod > div:first-child, .two > div > div.xevlxbw {' +
				'    position: relative !important;' +
				'    width: var(--wa-chatlist-width, 380px) !important;' +
				'    min-width: 220px !important;' +
				'    max-width: min(85vw, calc(100vw - 260px)) !important;' +
				'    flex: 0 0 var(--wa-chatlist-width, 380px) !important;' +
				'    height: 100% !important;' +
				'    transition: none !important;' +
				'  }' +
				'  #side, #main, .two > div { transition: none !important; }' +
				'  #side { width: 100% !important; max-width: 100% !important; height: 100% !important; flex: 1 1 auto !important; display: flex !important; flex-direction: column !important; }' +
				'  #pane-side, div[data-testid="chat-list"] { min-width: 200px !important; -webkit-overflow-scrolling: touch !important; }' +
				'  #main { min-width: 240px !important; -webkit-overflow-scrolling: touch !important; }' +
				'}' +
				'@media screen and (max-width: 640px) {' +
				'  #pane-side, div[data-testid="chat-list"], #main { min-width: 0 !important; }' +
				'}' +
				'#wa-side-resizer { position: absolute; top: 0; right: -3px; width: 7px; height: 100%; cursor: col-resize !important; z-index: 99999; user-select: none; -webkit-user-select: none; touch-action: none; background: transparent; transition: background-color 0.15s ease, box-shadow 0.15s ease; }' +
				'#wa-side-resizer:hover, #wa-side-resizer.wa-dragging { background-color: #00a884 !important; box-shadow: 0 0 8px rgba(0, 168, 132, 0.8) !important; }' +
				'body.wa-resizing { cursor: col-resize !important; user-select: none !important; -webkit-user-select: none !important; }' +
				'body.wa-resizing * { cursor: col-resize !important; user-select: none !important; -webkit-user-select: none !important; pointer-events: none !important; }' +
				'body.wa-resizing #wa-side-resizer { pointer-events: auto !important; }';

			var respTimer = null;
			function injectResponsive() {
				if (document.head && !document.getElementById('whatsapp-desktop-responsive')) {
					document.head.appendChild(respStyle);
					if (respTimer) {
						clearInterval(respTimer);
						respTimer = null;
					}
				}
			}
			injectResponsive();
			document.addEventListener('DOMContentLoaded', injectResponsive);
			window.addEventListener('load', injectResponsive);
			respTimer = setInterval(injectResponsive, 2500);
		})();

		// Draggable Chat List Resizer Controller
		(function() {
			var STORAGE_KEY = 'wa_chatlist_custom_width_v1';
			var DEFAULT_WIDTH = 380;
			var MIN_WIDTH = 220;

			function getMaxWidth() {
				return Math.max(MIN_WIDTH, window.innerWidth - 260);
			}

			function getStoredWidth() {
				var v = localStorage.getItem(STORAGE_KEY);
				if (v) {
					var parsed = parseInt(v, 10);
					if (!isNaN(parsed) && parsed >= MIN_WIDTH) {
						return Math.min(parsed, getMaxWidth());
					}
				}
				return DEFAULT_WIDTH;
			}

			var currentWidth = getStoredWidth();

			function getSidebarColumn() {
				var side = document.getElementById('side');
				if (!side) return null;
				var col = side.parentElement;
				if (col && !col.id) {
					col.id = 'wa-side-column';
				}
				return col;
			}

			var cachedBgCol = null;
			function getBackgroundColumn() {
				if (cachedBgCol && cachedBgCol.isConnected) return cachedBgCol;
				var bgChild = document.querySelector('.two > div.x47corl > div:first-child') ||
							  document.querySelector('.two > div > div.xevlxbw');
				if (bgChild) {
					cachedBgCol = bgChild;
					return bgChild;
				}
				var two = document.querySelector('.two');
				if (two) {
					for (var i = 0; i < two.children.length; i++) {
						var child = two.children[i];
						if (child.id === 'wa-side-column') continue;
						if (child.children && child.children.length === 2) {
							var c1 = child.children[0];
							var c2 = child.children[1];
							var cs2 = window.getComputedStyle(c2);
							if (cs2.borderLeftWidth && parseFloat(cs2.borderLeftWidth) > 0 && cs2.borderLeftStyle !== 'none') {
								cachedBgCol = c1;
								return c1;
							}
						}
					}
				}
				return null;
			}

			function applyWidthDirect(w, colNode, bgNode) {
				var wStr = w + 'px';
				document.documentElement.style.setProperty('--wa-chatlist-width', wStr);

				var col = colNode || getSidebarColumn();
				if (col) {
					col.style.setProperty('width', wStr, 'important');
					col.style.setProperty('flex', '0 0 ' + wStr, 'important');
					col.style.setProperty('max-width', wStr, 'important');
				}

				var bgCol = bgNode || getBackgroundColumn();
				if (bgCol) {
					bgCol.style.setProperty('width', wStr, 'important');
					bgCol.style.setProperty('flex', '0 0 ' + wStr, 'important');
					bgCol.style.setProperty('max-width', wStr, 'important');
				}
			}

			function setChatListWidth(width, save) {
				currentWidth = Math.max(MIN_WIDTH, Math.min(width, getMaxWidth()));
				if (save) {
					localStorage.setItem(STORAGE_KEY, currentWidth.toString());
				}

				applyWidthDirect(currentWidth);

				var side = document.getElementById('side');
				if (side) {
					side.style.setProperty('width', '100%', 'important');
					side.style.setProperty('max-width', '100%', 'important');
					side.style.setProperty('height', '100%', 'important');
					side.style.setProperty('flex', '1 1 auto', 'important');
				}

				var slider = document.getElementById('wa-chatlist-slider-input');
				var sliderVal = document.getElementById('wa-chatlist-slider-val');
				if (slider) slider.value = currentWidth;
				if (sliderVal) sliderVal.textContent = currentWidth + 'px';

				// Only notify React virtualization when resize is finalized, not during rapid dragging
				if (save) {
					window.dispatchEvent(new Event('resize'));
				}
			}

			window.setChatListWidth = setChatListWidth;
			window.getChatListWidth = function() { return currentWidth; };

			function injectResizerHandle() {
				var col = getSidebarColumn();
				if (!col) return;

				setChatListWidth(currentWidth, false);

				if (document.getElementById('wa-side-resizer')) return;

				var resizer = document.createElement('div');
				resizer.id = 'wa-side-resizer';
				resizer.title = 'Tarik untuk mengatur lebar list chat (Klik ganda untuk reset)';

				var isDragging = false;
				var startX = 0;
				var startWidth = 0;
				var dragRaf = null;
				var pendingWidth = 0;
				var cachedCol = null;
				var cachedBg = null;

				resizer.addEventListener('mousedown', function(e) {
					if (e.button !== 0) return;
					e.preventDefault();
					e.stopPropagation();

					isDragging = true;
					startX = e.clientX;
					cachedCol = getSidebarColumn();
					cachedBg = getBackgroundColumn();
					startWidth = cachedCol ? cachedCol.getBoundingClientRect().width : currentWidth;
					pendingWidth = startWidth;

					resizer.classList.add('wa-dragging');
					document.body.classList.add('wa-resizing');

					var onMouseMove = function(ev) {
						if (!isDragging) return;
						ev.preventDefault();
						var deltaX = ev.clientX - startX;
						pendingWidth = Math.max(MIN_WIDTH, Math.min(startWidth + deltaX, getMaxWidth()));
						if (!dragRaf) {
							dragRaf = requestAnimationFrame(function() {
								dragRaf = null;
								if (isDragging) {
									currentWidth = pendingWidth;
									applyWidthDirect(pendingWidth, cachedCol, cachedBg);
								}
							});
						}
					};

					var onMouseUp = function(ev) {
						if (!isDragging) return;
						isDragging = false;
						if (dragRaf) {
							cancelAnimationFrame(dragRaf);
							dragRaf = null;
						}
						resizer.classList.remove('wa-dragging');
						document.body.classList.remove('wa-resizing');
						window.removeEventListener('mousemove', onMouseMove, true);
						window.removeEventListener('mouseup', onMouseUp, true);

						var finalDelta = ev.clientX - startX;
						setChatListWidth(startWidth + finalDelta, true);

						cachedCol = null;
						cachedBg = null;
					};

					window.addEventListener('mousemove', onMouseMove, true);
					window.addEventListener('mouseup', onMouseUp, true);
				});

				resizer.addEventListener('dblclick', function(e) {
					e.preventDefault();
					e.stopPropagation();
					setChatListWidth(DEFAULT_WIDTH, true);
					if (window.showFloatingToast) {
						window.showFloatingToast('📐 Lebar list chat di-reset ke default (' + DEFAULT_WIDTH + 'px)');
					}
				});

				col.appendChild(resizer);
			}

			window.addEventListener('keydown', function(e) {
				if (e.metaKey || e.ctrlKey) {
					if (e.key === '[' || e.key === '{') {
						e.preventDefault();
						setChatListWidth(currentWidth - 30, true);
						if (window.showFloatingToast) {
							window.showFloatingToast('📐 Lebar list chat: ' + currentWidth + 'px');
						}
					} else if (e.key === ']' || e.key === '}') {
						e.preventDefault();
						setChatListWidth(currentWidth + 30, true);
						if (window.showFloatingToast) {
							window.showFloatingToast('📐 Lebar list chat: ' + currentWidth + 'px');
						}
					} else if (e.key === '\\' || e.key === '|') {
						e.preventDefault();
						setChatListWidth(DEFAULT_WIDTH, true);
						if (window.showFloatingToast) {
							window.showFloatingToast('📐 Lebar list chat di-reset (' + DEFAULT_WIDTH + 'px)');
						}
					}
				}
			});

			injectResizerHandle();
			document.addEventListener('DOMContentLoaded', injectResizerHandle);
			window.addEventListener('load', injectResizerHandle);
			setInterval(injectResizerHandle, 1500);

			var observer = new MutationObserver(function() {
				if (!document.getElementById('wa-side-resizer') && document.getElementById('side')) {
					injectResizerHandle();
				}
			});
			if (document.body) {
				observer.observe(document.body, { childList: true, subtree: true });
			} else {
				document.addEventListener('DOMContentLoaded', function() {
					observer.observe(document.body, { childList: true, subtree: true });
				});
			}

			window.addEventListener('resize', function() {
				if (currentWidth > getMaxWidth()) {
					setChatListWidth(getMaxWidth(), false);
				}
			});
		})();

		// Automatic Download & Document Preview Interceptor for Chat Files & Media
		(function() {
			function isDocumentFileName(name) {
				if (!name) return false;
				var ext = name.toLowerCase();
				return ext.endsWith('.pdf') || ext.endsWith('.doc') || ext.endsWith('.docx') ||
					   ext.endsWith('.xls') || ext.endsWith('.xlsx') || ext.endsWith('.ppt') ||
					   ext.endsWith('.pptx') || ext.endsWith('.txt') || ext.endsWith('.csv') ||
					   ext.endsWith('.rtf');
			}

			function captureDownload(href, filename, shouldAutoOpen) {
				if (!filename) filename = 'whatsapp_file';
				var isDoc = isDocumentFileName(filename);
				if (shouldAutoOpen === undefined) {
					shouldAutoOpen = isDoc;
				}
				showFloatingToast(isDoc ? ('📄 Opening preview: ' + filename + '...') : ('⏳ Downloading: ' + filename + '...'));

				fetch(href)
					.then(function(response) {
						return response.blob();
					})
					.then(function(blob) {
						var isPdf = filename.toLowerCase().endsWith('.pdf');
						var previewBlob = isPdf ? blob.slice(0, blob.size, 'application/pdf') : blob;
						var ownedBlobUrl = isPdf ? origCreateObjectURL(previewBlob) : '';
						var reader = new FileReader();
						reader.onloadend = function() {
							var base64data = reader.result;
							if (window.saveDownloadedFileNative) {
								window.saveDownloadedFileNative(filename, base64data).then(function(savedPath) {
									if (savedPath) {
										if (shouldAutoOpen) {
											showInAppDocModal(filename, ownedBlobUrl || href, savedPath, base64data, ownedBlobUrl);
											if (window.dismissStuckViewer) window.dismissStuckViewer();
											showFloatingToast('📄 Preview opened: ' + filename);
										} else {
											showFloatingToast('💾 Saved successfully: ' + filename);
										}
									} else {
										if (ownedBlobUrl) URL.revokeObjectURL(ownedBlobUrl);
										showFloatingToast('❌ Failed to save file.');
									}
								}).catch(function() {
									if (ownedBlobUrl) URL.revokeObjectURL(ownedBlobUrl);
									showFloatingToast('❌ Error saving file.');
								});
							}
						};
						reader.readAsDataURL(blob);
					})
					.catch(function(err) {
						console.error('Download intercept fetch error:', err);
					});
			}

			var forwardingDocumentDownload = false;
			var pendingViewerDownloadClick = false;
			var viewerDownloadSelector = [
				'button[data-testid*="download"]',
				'[role="button"][data-testid*="download"]',
				'button[aria-label*="Download" i]',
				'button[aria-label*="Unduh" i]',
				'[role="button"][aria-label*="Download" i]',
				'[role="button"][aria-label*="Unduh" i]',
				'button[title*="Download" i]',
				'button[title*="Unduh" i]',
				'[data-icon="download"]',
				'[data-icon="download-refreshed"]',
				'[data-icon*="download"]'
			].join(',');

			function findVisibleViewerDownloadControl() {
				var candidates = document.querySelectorAll(viewerDownloadSelector);
				var best = null;
				var bestScore = -1;
				for (var i = 0; i < candidates.length; i++) {
					var raw = candidates[i];
					if (raw.closest && raw.closest('#wa-doc-modal-overlay')) continue;
					var control = (raw.closest && raw.closest('button, a, [role="button"]')) || raw;
					var rect = control.getBoundingClientRect();
					if (rect.width < 8 || rect.height < 8 || rect.bottom <= 0 || rect.right <= 0 ||
						rect.top >= window.innerHeight || rect.left >= window.innerWidth) continue;
					var style = window.getComputedStyle(control);
					if (style.display === 'none' || style.visibility === 'hidden' || Number(style.opacity) === 0) continue;

					var score = 0;
					if (rect.top < window.innerHeight * 0.3) score += 4;
					if (rect.left > window.innerWidth * 0.55) score += 3;
					if (control.closest && control.closest('[role="dialog"], [data-testid*="viewer"], header, [role="toolbar"]')) score += 5;
					if (score > bestScore) {
						best = control;
						bestScore = score;
					}
				}
				return bestScore >= 4 ? best : null;
			}

			function triggerVisibleViewerDownload() {
				if (pendingViewerDownloadClick || !isRecentPDFIntent()) return false;
				var control = findVisibleViewerDownloadControl();
				if (!control) return false;
				pendingViewerDownloadClick = true;
				control.click();
				setTimeout(function() { pendingViewerDownloadClick = false; }, 1500);
				return true;
			}

			function findDocumentDownloadControl(start) {
				var selector = 'a[download], button[data-testid*="download"], [role="button"][data-testid*="download"], button[aria-label*="Unduh"], button[aria-label*="Download"], [role="button"][aria-label*="Unduh"], [role="button"][aria-label*="Download"], [data-icon="download"], [data-icon="download-refreshed"]';
				var node = start;
				for (var depth = 0; node && node !== document.body && depth < 12; depth++, node = node.parentElement) {
					var found = node.querySelector && node.querySelector(selector);
					if (found) return found.closest('button, a, [role="button"]') || found;
				}
				return null;
			}

			// Hook 1: Override HTMLAnchorElement.prototype.click (programmatic downloads)
			var originalAnchorClick = HTMLAnchorElement.prototype.click;
			HTMLAnchorElement.prototype.click = function() {
				var downloadAttr = this.getAttribute('download');
				var href = this.href || this.getAttribute('href');
				if ((downloadAttr !== null || this.download) && href && (href.indexOf('blob:') === 0 || href.indexOf('data:') === 0)) {
					var name = downloadAttr || this.download || lastClickedDocName || 'whatsapp_media';
					captureDownload(href, name, isDocumentFileName(name));
					return;
				}
				return originalAnchorClick.apply(this, arguments);
			};

			// Hook 2: User click event capturing (direct clicks on <a> with download)
			document.addEventListener('click', function(e) {
				var target = e.target;
				while (target && target !== document.body) {
					if (target.tagName === 'A') {
						var downloadAttr = target.getAttribute('download');
						var href = target.href || target.getAttribute('href');
						if ((downloadAttr !== null || target.download) && href && (href.indexOf('blob:') === 0 || href.indexOf('data:') === 0)) {
							e.preventDefault();
							e.stopPropagation();
							var name = downloadAttr || target.download || lastClickedDocName || 'whatsapp_media';
							captureDownload(href, name, isDocumentFileName(name));
							return;
						}
					}
					target = target.parentElement;
				}
			}, true);

			// Hook 3: Watch document bubble clicks in chat to handle viewer spinner
			document.addEventListener('click', function(e) {
				if (forwardingDocumentDownload) return;
				var el = e.target;
				if (!el) return;

				// Completely ignore clicks inside media-viewer or custom modal overlay
				if (typeof el.closest === 'function') {
					if (el.closest('[data-testid="media-viewer"]') || el.closest('#wa-doc-modal-overlay')) {
						return;
					}
				}

				var foundName = extractDocumentName(el);
				var clickedDoc = !!foundName;

				if (clickedDoc) {
					lastClickedDocName = foundName;
					lastDocumentIntentAt = Date.now();
					if (isDocumentFileName(foundName)) {
						var directDownload = findDocumentDownloadControl(el);
						if (directDownload && !directDownload.contains(el)) {
							e.preventDefault();
							e.stopImmediatePropagation();
							forwardingDocumentDownload = true;
							directDownload.click();
							forwardingDocumentDownload = false;
							return;
						}
					}

					var checkCount = 0;
					var checkTimer = setInterval(function() {
						checkCount++;
						if (checkCount > 30) {
							clearInterval(checkTimer);
							return;
						}

						if (triggerVisibleViewerDownload()) {
							clearInterval(checkTimer);
						}
					}, 200);
				}
			}, true);

			// Hook 4: MutationObserver to auto-dismiss stuck media viewer and trigger download/preview
			var viewerObserver = new MutationObserver(function() {
				if (!isRecentPDFIntent()) return;
				if (!document.getElementById('wa-doc-modal-overlay')) triggerVisibleViewerDownload();
			});

			function initViewerObserver() {
				var target = document.body || document.documentElement;
				if (target) {
					viewerObserver.observe(target, { childList: true, subtree: true });
				} else {
					document.addEventListener('DOMContentLoaded', initViewerObserver, { once: true });
				}
			}
			initViewerObserver();
		})();

		// Theme Manager, In-Flow Header Toolbar Button & Control Center Modal
		(function() {
			var isMac = navigator.platform.toUpperCase().indexOf('MAC') >= 0;
			var currentTheme = 'dark';
			var themeObserver = null;

			// --- Override window.matchMedia for prefers-color-scheme ---
			var origMatchMedia = window.matchMedia;
			if (origMatchMedia) {
				window.matchMedia = function(query) {
					var res = origMatchMedia.apply(this, arguments);
					if (query && query.indexOf('prefers-color-scheme') >= 0) {
						var isDarkQuery = query.indexOf('dark') >= 0;
						var forcedMatches = isDarkQuery;
						if (currentTheme === 'light') {
							forcedMatches = !isDarkQuery;
						} else if (currentTheme === 'dark') {
							forcedMatches = isDarkQuery;
						} else {
							forcedMatches = res.matches;
						}

						return {
							matches: forcedMatches,
							media: query,
							addEventListener: function(t, fn) { res.addEventListener ? res.addEventListener(t, fn) : (res.addListener && res.addListener(fn)); },
							removeEventListener: function(t, fn) { res.removeEventListener ? res.removeEventListener(t, fn) : (res.removeListener && res.removeListener(fn)); },
							addListener: function(fn) { if (res.addListener) res.addListener(fn); },
							removeListener: function(fn) { if (res.removeListener) res.removeListener(fn); },
							onchange: null
						};
					}
					return res;
				};
			}

			// --- Theme Management ---
			function getSystemIsDark() {
				if (origMatchMedia) {
					return origMatchMedia.call(window, '(prefers-color-scheme: dark)').matches;
				}
				return true;
			}

			function applyThemeToDOM(theme) {
				currentTheme = theme;
				var isDark = (theme === 'system') ? getSystemIsDark() : (theme === 'dark');

				// 1. Update documentElement & body classes
				if (isDark) {
					document.documentElement.classList.add('dark');
					document.documentElement.classList.remove('light');
					if (document.body) {
						document.body.classList.add('dark');
						document.body.classList.remove('light');
					}
					document.documentElement.setAttribute('data-theme', 'dark');
					document.documentElement.style.colorScheme = 'dark';
				} else {
					document.documentElement.classList.remove('dark');
					document.documentElement.classList.add('light');
					if (document.body) {
						document.body.classList.remove('dark');
						document.body.classList.add('light');
					}
					document.documentElement.setAttribute('data-theme', 'light');
					document.documentElement.style.colorScheme = 'light';
				}

				// 2. Synchronize WhatsApp Web's own localStorage keys
				try {
					if (theme === 'system') {
						localStorage.setItem('system-theme-mode', 'true');
						localStorage.setItem('theme', JSON.stringify(isDark ? 'dark' : 'light'));
					} else {
						localStorage.setItem('system-theme-mode', 'false');
						localStorage.setItem('theme', JSON.stringify(theme));
					}
				} catch(e) {}

				// 3. Update modal if visible
				if (window.syncModalTheme) {
					window.syncModalTheme(isDark);
				}

				// 4. Ensure MutationObserver prevents WhatsApp from reverting body theme class
				if (window.MutationObserver && document.body) {
					if (!themeObserver) {
						themeObserver = new MutationObserver(function() {
							var shouldBeDark = (currentTheme === 'system') ? getSystemIsDark() : (currentTheme === 'dark');
							if (shouldBeDark && !document.body.classList.contains('dark')) {
								document.body.classList.add('dark');
							} else if (!shouldBeDark && document.body.classList.contains('dark')) {
								document.body.classList.remove('dark');
							}
						});
					}
					themeObserver.disconnect();
					themeObserver.observe(document.body, { attributes: true, attributeFilter: ['class'] });
				}
			}

			window.getAppTheme = function() {
				return currentTheme;
			};

			window.setAppTheme = function(theme) {
				if (theme !== 'dark' && theme !== 'light' && theme !== 'system') {
					theme = 'dark';
				}
				applyThemeToDOM(theme);
				if (window.setAppThemeNative) {
					window.setAppThemeNative(theme);
				}
				showFloatingToast(theme === 'dark' ? '🌙 Theme: Dark Mode' : (theme === 'light' ? '☀️ Theme: Light Mode' : '💻 Theme: Follow System'));
			};

			// Listen for system appearance changes
			if (origMatchMedia) {
				var sysMedia = origMatchMedia.call(window, '(prefers-color-scheme: dark)');
				var onSysChange = function() {
					if (currentTheme === 'system') {
						applyThemeToDOM('system');
					}
				};
				if (sysMedia.addEventListener) {
					sysMedia.addEventListener('change', onSysChange);
				} else if (sysMedia.addListener) {
					sysMedia.addListener(onSysChange);
				}
			}

			// Load saved theme from native settings and keep synced
			function initTheme() {
				if (window.getAppThemeNative) {
					window.getAppThemeNative().then(function(savedTheme) {
						if (savedTheme) {
							applyThemeToDOM(savedTheme);
						}
					});
				}
			}
			initTheme();
			document.addEventListener('DOMContentLoaded', initTheme);
			window.addEventListener('load', initTheme);
			setInterval(function() {
				if (document.body && !themeObserver) {
					applyThemeToDOM(currentTheme);
				}
			}, 2000);

			// --- In-Flow Header Toolbar Button (Non-Floating, Clean WhatsApp Style) ---
			function injectHeaderToolbarBtn() {
				if (document.getElementById('wa-toolbar-settings-btn')) return;

				// Target WhatsApp Web's left header above chats
				var header = document.querySelector('#side header') || document.querySelector('header');
				if (!header) return;

				// Find actions container inside header (where Status, Channels, New Chat icons live)
				var actionsWrap = header.querySelector('div:last-child') || header.querySelector('span:last-child') || header;
				if (!actionsWrap) return;

				var btn = document.createElement('button');
				btn.id = 'wa-toolbar-settings-btn';
				btn.setAttribute('aria-label', 'Settings & Controls');
				btn.title = 'Settings & Controls (' + (isMac ? 'Cmd' : 'Ctrl') + ' + ,)';
				btn.style.cssText = 'width:40px;height:40px;border-radius:50%;display:inline-flex;align-items:center;justify-content:center;background:transparent;border:none;color:#aebac1;cursor:pointer;outline:none;transition:background-color 0.15s ease, color 0.15s ease;flex-shrink:0;margin:0 2px;';
				btn.innerHTML = '<svg width="20" height="20" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.8" stroke-linecap="round" stroke-linejoin="round">' +
					'<circle cx="12" cy="12" r="3"></circle>' +
					'<path d="M19.4 15a1.65 1.65 0 0 0 .33 1.82l.06.06a2 2 0 0 1 0 2.83 2 2 0 0 1-2.83 0l-.06-.06a1.65 1.65 0 0 0-1.82-.33 1.65 1.65 0 0 0-1 1.51V21a2 2 0 0 1-2 2 2 2 0 0 1-2-2v-.09A1.65 1.65 0 0 0 9 19.4a1.65 1.65 0 0 0-1.82.33l-.06.06a2 2 0 0 1-2.83 0 2 2 0 0 1 0-2.83l.06-.06a1.65 1.65 0 0 0 .33-1.82 1.65 1.65 0 0 0-1.51-1H3a2 2 0 0 1-2-2 2 2 0 0 1 2-2h.09A1.65 1.65 0 0 0 4.6 9a1.65 1.65 0 0 0-.33-1.82l-.06-.06a2 2 0 0 1 0-2.83 2 2 0 0 1 2.83 0l.06.06a1.65 1.65 0 0 0 1.82.33H9a1.65 1.65 0 0 0 1-1.51V3a2 2 0 0 1 2-2 2 2 0 0 1 2 2v.09a1.65 1.65 0 0 0 1 1.51 1.65 1.65 0 0 0 1.82-.33l.06-.06a2 2 0 0 1 2.83 0 2 2 0 0 1 0 2.83l-.06.06a1.65 1.65 0 0 0-.33 1.82V9a1.65 1.65 0 0 0 1.51 1H21a2 2 0 0 1 2 2 2 2 0 0 1-2 2h-.09a1.65 1.65 0 0 0-1.51 1z"></path>' +
					'</svg>';

				btn.onmouseenter = function() {
					btn.style.backgroundColor = document.body.classList.contains('dark') ? 'rgba(255,255,255,0.08)' : 'rgba(0,0,0,0.06)';
					btn.style.color = document.body.classList.contains('dark') ? '#e9edef' : '#111b21';
				};
				btn.onmouseleave = function() {
					btn.style.backgroundColor = 'transparent';
					btn.style.color = '#aebac1';
				};
				btn.onclick = function(e) {
					e.stopPropagation();
					window.showSettingsModal();
				};

				actionsWrap.appendChild(btn);
			}

			injectHeaderToolbarBtn();
			document.addEventListener('DOMContentLoaded', injectHeaderToolbarBtn);
			window.addEventListener('load', injectHeaderToolbarBtn);
			setInterval(injectHeaderToolbarBtn, 2000);

			// --- Minimalist WhatsApp Control Center Modal ---
			window.showSettingsModal = function() {
				if (document.getElementById('wa-settings-overlay')) {
					var ex = document.getElementById('wa-settings-overlay');
					if (ex.parentNode) ex.parentNode.removeChild(ex);
					return;
				}

				var isDark = currentTheme === 'system' ?
					(window.matchMedia && window.matchMedia('(prefers-color-scheme: dark)').matches) :
					(currentTheme === 'dark');

				var overlay = document.createElement('div');
				overlay.id = 'wa-settings-overlay';
				overlay.style.cssText = 'position:fixed;inset:0;background:rgba(8,15,19,.68);z-index:9999999;display:flex;align-items:center;justify-content:center;padding:16px;box-sizing:border-box;font-family:-apple-system,BlinkMacSystemFont,"Segoe UI",system-ui,sans-serif;';

				var modal = document.createElement('div');
				modal.id = 'wa-settings-container';
				modal.style.cssText = 'width:520px;max-width:96vw;max-height:90vh;border-radius:10px;box-sizing:border-box;display:flex;flex-direction:column;gap:0;overflow-y:auto;padding:0 22px 18px;box-shadow:0 18px 48px rgba(0,0,0,.32);';

				// Header
				var header = document.createElement('div');
				header.id = 'wa-modal-header';
				header.style.cssText = 'display:flex;align-items:center;justify-content:space-between;border-bottom-width:1px;border-bottom-style:solid;padding:18px 0 14px;margin-bottom:2px;';
				header.innerHTML = '' +
					'<div style="display:flex;align-items:center;gap:10px;">' +
					'  <div id="wa-modal-icon-wrap" style="width:10px;height:10px;border-radius:50%;display:flex;align-items:center;justify-content:center;background:#00a884;">' +
					'  </div>' +
					'  <div>' +
					'    <h3 id="wa-modal-title" style="margin:0;font-size:15px;font-weight:600;">WhatsApp Desk</h3>' +
					'    <span id="wa-modal-sub" style="font-size:11px;">Application settings · version 1.5.4</span>' +
					'  </div>' +
					'</div>' +
					'<button id="wa-settings-close-x" style="background:transparent;border:none;cursor:pointer;font-size:18px;line-height:1;padding:4px 8px;border-radius:4px;">✕</button>';
				modal.appendChild(header);

				// Section 0: Theme Switcher Segmented Control
				var themeBox = document.createElement('div');
				themeBox.className = 'wa-modal-card';
				themeBox.style.cssText = 'display:flex;align-items:center;justify-content:space-between;padding:14px 0;border-radius:0;border-width:0 0 1px;border-style:solid;gap:16px;';
				themeBox.innerHTML = '' +
					'<div>' +
					'  <strong class="wa-text-primary" style="font-size:12.5px;display:block;">Appearance</strong>' +
					'  <span class="wa-text-muted" style="font-size:11px;">Application interface theme</span>' +
					'</div>' +
					'<div style="display:flex;align-items:center;gap:4px;">' +
					'  <button id="wa-theme-btn-dark" class="wa-theme-btn" style="padding:5px 10px;border-radius:6px;font-size:11.5px;cursor:pointer;border-width:1px;border-style:solid;font-weight:500;">Dark</button>' +
					'  <button id="wa-theme-btn-light" class="wa-theme-btn" style="padding:5px 10px;border-radius:6px;font-size:11.5px;cursor:pointer;border-width:1px;border-style:solid;font-weight:500;">Light</button>' +
					'  <button id="wa-theme-btn-system" class="wa-theme-btn" style="padding:5px 10px;border-radius:6px;font-size:11.5px;cursor:pointer;border-width:1px;border-style:solid;font-weight:500;">System</button>' +
					'</div>';
				modal.appendChild(themeBox);

				// Section 1: Quick Interactive Controls (2-Column Grid)
				var quickGrid = document.createElement('div');
				quickGrid.style.cssText = 'display:flex;flex-direction:column;';

				// Card 1: Privacy Mode
				var cardPrivacy = document.createElement('div');
				cardPrivacy.className = 'wa-modal-card';
				cardPrivacy.style.cssText = 'border-radius:0;border-width:0 0 1px;border-style:solid;padding:12px 0;display:flex;align-items:center;justify-content:space-between;gap:16px;';
				cardPrivacy.innerHTML = '' +
					'<div>' +
					'  <div style="display:flex;align-items:center;justify-content:space-between;margin-bottom:2px;">' +
					'    <strong class="wa-text-primary" style="font-size:12.5px;">Privacy Mode</strong>' +
					'    <span id="wa-badge-priv" style="font-size:10px;padding:1px 5px;border-radius:4px;font-weight:600;">...</span>' +
					'  </div>' +
					'  <div class="wa-text-muted" style="font-size:11px;">Blur chats and media when cursor is idle.</div>' +
					'</div>' +
					'<div style="display:flex;align-items:center;justify-content:space-between;">' +
					'  <span class="wa-text-muted" style="font-size:10px;font-family:monospace;">' + (isMac ? 'Cmd' : 'Ctrl') + '+Shift+P</span>' +
					'  <button id="wa-action-toggle-priv" class="wa-card-btn" style="padding:4px 10px;border-radius:6px;font-size:11px;font-weight:600;cursor:pointer;border-width:1px;border-style:solid;">Toggle</button>' +
					'</div>';
				quickGrid.appendChild(cardPrivacy);

				// Card 2: Always on Top
				var cardPin = document.createElement('div');
				cardPin.className = 'wa-modal-card';
				cardPin.style.cssText = 'border-radius:0;border-width:0 0 1px;border-style:solid;padding:12px 0;display:flex;align-items:center;justify-content:space-between;gap:16px;';
				cardPin.innerHTML = '' +
					'<div>' +
					'  <div style="display:flex;align-items:center;justify-content:space-between;margin-bottom:2px;">' +
					'    <strong class="wa-text-primary" style="font-size:12.5px;">Always on Top</strong>' +
					'    <span id="wa-badge-pin" style="font-size:10px;padding:1px 5px;border-radius:4px;font-weight:600;">...</span>' +
					'  </div>' +
					'  <div class="wa-text-muted" style="font-size:11px;">Keep window floating above other applications.</div>' +
					'</div>' +
					'<div style="display:flex;align-items:center;justify-content:space-between;">' +
					'  <span class="wa-text-muted" style="font-size:10px;font-family:monospace;">' + (isMac ? 'Cmd' : 'Ctrl') + '+Shift+T</span>' +
					'  <button id="wa-action-toggle-pin" class="wa-card-btn" style="padding:4px 10px;border-radius:6px;font-size:11px;font-weight:600;cursor:pointer;border-width:1px;border-style:solid;">Toggle</button>' +
					'</div>';
				quickGrid.appendChild(cardPin);

				// Card 3: Audio Mute
				var cardMute = document.createElement('div');
				cardMute.className = 'wa-modal-card';
				cardMute.style.cssText = 'border-radius:0;border-width:0 0 1px;border-style:solid;padding:12px 0;display:flex;align-items:center;justify-content:space-between;gap:16px;';
				cardMute.innerHTML = '' +
					'<div>' +
					'  <div style="display:flex;align-items:center;justify-content:space-between;margin-bottom:2px;">' +
					'    <strong class="wa-text-primary" style="font-size:12.5px;">Notification Audio</strong>' +
					'    <span id="wa-badge-mute" style="font-size:10px;padding:1px 5px;border-radius:4px;font-weight:600;">...</span>' +
					'  </div>' +
					'  <div class="wa-text-muted" style="font-size:11px;">Mute all notification sounds and media audio.</div>' +
					'</div>' +
					'<div style="display:flex;align-items:center;justify-content:space-between;">' +
					'  <span class="wa-text-muted" style="font-size:10px;font-family:monospace;">' + (isMac ? 'Cmd' : 'Ctrl') + '+Shift+M</span>' +
					'  <button id="wa-action-toggle-mute" class="wa-card-btn" style="padding:4px 10px;border-radius:6px;font-size:11px;font-weight:600;cursor:pointer;border-width:1px;border-style:solid;">Toggle</button>' +
					'</div>';
				quickGrid.appendChild(cardMute);

				// Card 4: Auto-Start
				var cardAuto = document.createElement('div');
				cardAuto.className = 'wa-modal-card';
				cardAuto.style.cssText = 'border-radius:0;border-width:0 0 1px;border-style:solid;padding:12px 0;display:flex;align-items:center;justify-content:space-between;gap:16px;';
				cardAuto.innerHTML = '' +
					'<div>' +
					'  <div style="display:flex;align-items:center;justify-content:space-between;margin-bottom:2px;">' +
					'    <strong class="wa-text-primary" style="font-size:12.5px;">Launch at Startup</strong>' +
					'    <span id="wa-badge-auto" style="font-size:10px;padding:1px 5px;border-radius:4px;font-weight:600;">...</span>' +
					'  </div>' +
					'  <div class="wa-text-muted" style="font-size:11px;">Automatically start WhatsApp Desk on system login.</div>' +
					'</div>' +
					'<div style="display:flex;align-items:center;justify-content:space-between;">' +
					'  <span class="wa-text-muted" style="font-size:10px;font-family:monospace;">' + (isMac ? 'Cmd' : 'Ctrl') + '+Shift+S</span>' +
					'  <button id="wa-action-toggle-auto" class="wa-card-btn" style="padding:4px 10px;border-radius:6px;font-size:11px;font-weight:600;cursor:pointer;border-width:1px;border-style:solid;">Toggle</button>' +
					'</div>';
				quickGrid.appendChild(cardAuto);

				modal.appendChild(quickGrid);

				// Section 1.5: Chat List Width Slider
				var widthSection = document.createElement('div');
				widthSection.className = 'wa-modal-card';
				widthSection.style.cssText = 'display:flex;flex-direction:column;gap:8px;border-radius:0;border-width:0 0 1px;border-style:solid;padding:14px 0;';
				var curChatW = window.getChatListWidth ? window.getChatListWidth() : 380;
				widthSection.innerHTML = '' +
					'<div style="display:flex;align-items:center;justify-content:space-between;">' +
					'  <div>' +
					'    <strong class="wa-text-primary" style="font-size:12.5px;display:block;">Chat List Width</strong>' +
					'    <span class="wa-text-muted" style="font-size:11px;">Adjust sidebar size (drag border or use slider)</span>' +
					'  </div>' +
					'  <div style="display:flex;align-items:center;gap:8px;">' +
					'    <span id="wa-chatlist-slider-val" style="font-size:11px;font-family:monospace;font-weight:600;color:#00a884;">' + curChatW + 'px</span>' +
					'    <button id="wa-btn-reset-width" class="wa-card-btn" style="padding:3px 8px;border-radius:5px;font-size:10.5px;cursor:pointer;border-width:1px;border-style:solid;">Reset</button>' +
					'  </div>' +
					'</div>' +
					'<div style="display:flex;align-items:center;gap:12px;margin-top:2px;">' +
					'  <span class="wa-text-muted" style="font-size:11px;">220px</span>' +
					'  <input id="wa-chatlist-slider-input" type="range" min="220" max="650" value="' + curChatW + '" style="flex:1;accent-color:#00a884;cursor:pointer;" />' +
					'  <span class="wa-text-muted" style="font-size:11px;">650px</span>' +
					'</div>' +
					'<div class="wa-text-muted" style="font-size:10px;">Shortcuts: <kbd style="font-family:monospace;">' + (isMac ? 'Cmd' : 'Ctrl') + ' + [</kbd> (shrink), <kbd style="font-family:monospace;">' + (isMac ? 'Cmd' : 'Ctrl') + ' + ]</kbd> (widen), <kbd style="font-family:monospace;">' + (isMac ? 'Cmd' : 'Ctrl') + ' + \\</kbd> (reset)</div>';
				modal.appendChild(widthSection);

				var sliderIn = widthSection.querySelector('#wa-chatlist-slider-input');
				var sliderVal = widthSection.querySelector('#wa-chatlist-slider-val');
				var btnResetW = widthSection.querySelector('#wa-btn-reset-width');
				if (sliderIn) {
					sliderIn.oninput = function() {
						var v = parseInt(sliderIn.value, 10);
						if (sliderVal) sliderVal.textContent = v + 'px';
						if (window.setChatListWidth) window.setChatListWidth(v, false);
					};
					sliderIn.onchange = function() {
						var v = parseInt(sliderIn.value, 10);
						if (window.setChatListWidth) window.setChatListWidth(v, true);
					};
				}
				if (btnResetW) {
					btnResetW.onclick = function() {
						if (window.setChatListWidth) window.setChatListWidth(380, true);
						if (sliderIn) sliderIn.value = '380';
						if (sliderVal) sliderVal.textContent = '380px';
					};
				}

				// Section 2: Download Folder Settings
				var folderSection = document.createElement('div');
				folderSection.className = 'wa-modal-card';
				folderSection.style.cssText = 'display:flex;flex-direction:column;gap:8px;border-radius:0;border-width:0 0 1px;border-style:solid;padding:14px 0;';
				folderSection.innerHTML = '' +
					'<div style="display:flex;align-items:center;justify-content:space-between;">' +
					'  <strong class="wa-text-primary" style="font-size:12.5px;">Downloads folder</strong>' +
					'  <button id="wa-btn-reset-folder" style="background:transparent;border:none;color:#00a884;font-size:11px;cursor:pointer;padding:2px 4px;">Use default</button>' +
					'</div>' +
					'<div class="wa-text-muted" style="font-size:11px;">Files & media downloaded from chat are permanently saved here:</div>' +
					'<div id="wa-folder-box" style="display:flex;align-items:center;border-width:1px;border-style:solid;border-radius:6px;padding:6px 8px;min-width:0;">' +
					'  <span id="wa-folder-path" style="font-size:11px;white-space:nowrap;overflow:hidden;text-overflow:ellipsis;flex:1;font-family:monospace;">Loading...</span>' +
					'</div>' +
					'<div style="display:flex;align-items:center;gap:6px;margin-top:2px;">' +
					'  <button id="wa-btn-change-folder" class="wa-card-btn" style="flex:1;padding:6px 10px;border-radius:6px;font-size:11.5px;font-weight:500;cursor:pointer;border-width:1px;border-style:solid;">Change Folder Location...</button>' +
					'  <button id="wa-btn-open-folder" style="background:#00a884;color:#111b21;border:none;padding:6px 12px;border-radius:6px;font-size:11.5px;font-weight:600;cursor:pointer;">' + (isMac ? 'Open in Finder' : 'Open Folder') + '</button>' +
					'</div>';
				modal.appendChild(folderSection);

				// Section 3: Maintenance & Update Actions
				var actionsSection = document.createElement('div');
				actionsSection.className = 'wa-modal-card';
				actionsSection.style.cssText = 'display:flex;flex-direction:column;gap:8px;border-radius:0;border-width:0 0 1px;border-style:solid;padding:14px 0;';
				actionsSection.innerHTML = '' +
					'<strong class="wa-text-primary" style="font-size:12.5px;">Maintenance & Tools</strong>' +
					'<div style="display:grid;grid-template-columns:1fr 1fr;gap:6px;">' +
					'  <button id="wa-btn-devtools-modal" class="wa-card-btn" style="padding:6px 8px;border-radius:6px;font-size:11.5px;font-weight:500;cursor:pointer;border-width:1px;border-style:solid;text-align:center;">DevTools & HUD</button>' +
					'  <button id="wa-btn-reload-modal" class="wa-card-btn" style="padding:6px 8px;border-radius:6px;font-size:11.5px;font-weight:500;cursor:pointer;border-width:1px;border-style:solid;text-align:center;">Reload chat</button>' +
					'  <button id="wa-btn-hardref-modal" class="wa-card-btn" style="padding:6px 8px;border-radius:6px;font-size:11.5px;font-weight:500;cursor:pointer;border-width:1px;border-style:solid;text-align:center;">Clear cache</button>' +
					'  <button id="wa-btn-onboard-modal" class="wa-card-btn" style="padding:6px 8px;border-radius:6px;font-size:11.5px;font-weight:500;cursor:pointer;border-width:1px;border-style:solid;text-align:center;">View welcome guide</button>' +
					'</div>';
				modal.appendChild(actionsSection);

				// Disclaimer
				var disclaimer = document.createElement('div');
				disclaimer.className = 'wa-text-muted';
				disclaimer.style.cssText = 'font-size:10px;line-height:1.4;border-top-width:1px;border-top-style:solid;padding-top:8px;margin-top:2px;';
				disclaimer.innerHTML = '<strong>WhatsApp Desk</strong> is an independent application and is not affiliated with Meta.';
				modal.appendChild(disclaimer);

				// Footer
				var footer = document.createElement('div');
				footer.style.cssText = 'display:flex;justify-content:space-between;align-items:center;margin-top:2px;';
				footer.innerHTML = '<span class="wa-text-muted" style="font-size:10.5px;">Press <kbd style="padding:1px 3px;border-radius:3px;font-family:monospace;">Esc</kbd> to close</span>';
				var btnDone = document.createElement('button');
				btnDone.textContent = 'Done';
				btnDone.id = 'wa-btn-done';
				btnDone.style.cssText = 'padding:5px 16px;border-radius:6px;font-size:11.5px;font-weight:600;cursor:pointer;border-width:1px;border-style:solid;';
				footer.appendChild(btnDone);
				modal.appendChild(footer);

				overlay.appendChild(modal);
				document.body.appendChild(overlay);
				modal.addEventListener('pointerdown', function(e) { e.stopPropagation(); });
				modal.addEventListener('click', function(e) { e.stopPropagation(); });

				function closeSettings() {
					window.removeEventListener('keydown', onKeyClose);
					window.syncModalTheme = null;
					if (overlay.parentNode) overlay.parentNode.removeChild(overlay);
				}
				function onKeyClose(e) {
					if (e.key === 'Escape') closeSettings();
				}
				window.addEventListener('keydown', onKeyClose);
				btnDone.onclick = closeSettings;
				document.getElementById('wa-settings-close-x').onclick = closeSettings;
				overlay.onclick = function(e) {
					if (e.target === overlay) closeSettings();
				};

				// Styling Synchronizer for Modal (Dark / Light Theme)
				window.syncModalTheme = function(isThemeDark) {
					var bg = isThemeDark ? '#111b21' : '#ffffff';
					var cardBg = bg;
					var border = isThemeDark ? '#2a3942' : '#d1d7db';
					var textPri = isThemeDark ? '#e9edef' : '#111b21';
					var textMut = isThemeDark ? '#8696a0' : '#667781';
					var accent = isThemeDark ? '#00a884' : '#008069';

					modal.style.background = bg;
					modal.style.border = '1px solid ' + border;
					header.style.borderBottomColor = border;
					document.getElementById('wa-modal-title').style.color = textPri;
					document.getElementById('wa-modal-sub').style.color = textMut;
					document.getElementById('wa-modal-icon-wrap').style.background = accent;
					document.getElementById('wa-modal-icon-wrap').style.color = accent;
					document.getElementById('wa-settings-close-x').style.color = textMut;

					document.querySelectorAll('.wa-modal-card').forEach(function(el) {
						el.style.background = cardBg;
						el.style.borderColor = border;
					});
					document.querySelectorAll('.wa-text-primary').forEach(function(el) {
						el.style.color = textPri;
					});
					document.querySelectorAll('.wa-text-muted').forEach(function(el) {
						el.style.color = textMut;
					});

					var fBox = document.getElementById('wa-folder-box');
					if (fBox) {
						fBox.style.background = isThemeDark ? '#111b21' : '#ffffff';
						fBox.style.borderColor = border;
					}
					var fPath = document.getElementById('wa-folder-path');
					if (fPath) fPath.style.color = textMut;

					var btnOpen = document.getElementById('wa-btn-open-folder');
					if (btnOpen) {
						btnOpen.style.background = accent;
						btnOpen.style.color = isThemeDark ? '#111b21' : '#ffffff';
					}

					document.querySelectorAll('.wa-card-btn').forEach(function(el) {
						el.style.background = isThemeDark ? '#111b21' : '#ffffff';
						el.style.borderColor = border;
						el.style.color = textPri;
					});

					btnDone.style.background = isThemeDark ? '#202c33' : '#e9edef';
					btnDone.style.borderColor = border;
					btnDone.style.color = textPri;

					// Theme segment buttons
					['dark', 'light', 'system'].forEach(function(mode) {
						var tBtn = document.getElementById('wa-theme-btn-' + mode);
						if (tBtn) {
							var active = (currentTheme === mode);
							tBtn.style.background = active ? accent : (isThemeDark ? '#111b21' : '#ffffff');
							tBtn.style.color = active ? (isThemeDark ? '#111b21' : '#ffffff') : textPri;
							tBtn.style.borderColor = active ? accent : border;
						}
					});
				};

				// Synchronize Toggle Badges & Button States
				function updateBadges() {
					var isThemeDark = currentTheme === 'system' ?
						(window.matchMedia && window.matchMedia('(prefers-color-scheme: dark)').matches) :
						(currentTheme === 'dark');
					var accent = isThemeDark ? '#00a884' : '#008069';

					var privActive = window.isPrivacyModeActive ? window.isPrivacyModeActive() : false;
					var badgePriv = document.getElementById('wa-badge-priv');
					var btnPriv = document.getElementById('wa-action-toggle-priv');
					if (badgePriv && btnPriv) {
						badgePriv.textContent = privActive ? 'Enabled' : 'Disabled';
						badgePriv.style.background = privActive ? (isThemeDark ? 'rgba(0,168,132,0.15)' : 'rgba(0,128,105,0.15)') : 'transparent';
						badgePriv.style.color = privActive ? accent : '#8696a0';
						btnPriv.textContent = privActive ? 'Disable' : 'Enable';
					}

					var pinActive = window.isAlwaysOnTopActive ? window.isAlwaysOnTopActive() : false;
					var badgePin = document.getElementById('wa-badge-pin');
					var btnPin = document.getElementById('wa-action-toggle-pin');
					if (badgePin && btnPin) {
						badgePin.textContent = pinActive ? 'Pinned' : 'Unpinned';
						badgePin.style.background = pinActive ? (isThemeDark ? 'rgba(0,168,132,0.15)' : 'rgba(0,128,105,0.15)') : 'transparent';
						badgePin.style.color = pinActive ? accent : '#8696a0';
						btnPin.textContent = pinActive ? 'Unpin' : 'Pin';
					}

					var muteActive = window.isAudioMuted ? window.isAudioMuted() : false;
					var badgeMute = document.getElementById('wa-badge-mute');
					var btnMute = document.getElementById('wa-action-toggle-mute');
					if (badgeMute && btnMute) {
						badgeMute.textContent = muteActive ? 'Muted' : 'Unmuted';
						badgeMute.style.background = muteActive ? 'rgba(234,0,56,0.15)' : 'transparent';
						badgeMute.style.color = muteActive ? '#ff5252' : accent;
						btnMute.textContent = muteActive ? 'Unmute' : 'Mute';
					}

					var autoActive = window.isAutoStartActive ? window.isAutoStartActive() : false;
					var badgeAuto = document.getElementById('wa-badge-auto');
					var btnAuto = document.getElementById('wa-action-toggle-auto');
					if (badgeAuto && btnAuto) {
						badgeAuto.textContent = autoActive ? 'Enabled' : 'Disabled';
						badgeAuto.style.background = autoActive ? (isThemeDark ? 'rgba(0,168,132,0.15)' : 'rgba(0,128,105,0.15)') : 'transparent';
						badgeAuto.style.color = autoActive ? accent : '#8696a0';
						btnAuto.textContent = autoActive ? 'Disable' : 'Enable';
					}

					window.syncModalTheme(isThemeDark);
				}
				updateBadges();

				// Hook Theme Segmented Control
				document.getElementById('wa-theme-btn-dark').onclick = function() {
					window.setAppTheme('dark');
					updateBadges();
				};
				document.getElementById('wa-theme-btn-light').onclick = function() {
					window.setAppTheme('light');
					updateBadges();
				};
				document.getElementById('wa-theme-btn-system').onclick = function() {
					window.setAppTheme('system');
					updateBadges();
				};

				// Hook Click Actions
				document.getElementById('wa-action-toggle-priv').onclick = function() {
					if (window.togglePrivacyMode) window.togglePrivacyMode();
					updateBadges();
				};
				document.getElementById('wa-action-toggle-pin').onclick = function() {
					if (window.toggleAlwaysOnTop) {
						window.toggleAlwaysOnTop().then(function() { updateBadges(); });
					}
				};
				document.getElementById('wa-action-toggle-mute').onclick = function() {
					if (window.toggleMuteAudio) window.toggleMuteAudio();
					updateBadges();
				};
				document.getElementById('wa-action-toggle-auto').onclick = function() {
					if (window.toggleAutoStart) {
						window.toggleAutoStart().then(function() { updateBadges(); });
					}
				};

				document.getElementById('wa-btn-devtools-modal').onclick = function() {
					closeSettings();
					if (window.ndWA && window.ndWA.toggleHUD) window.ndWA.toggleHUD();
				};
				document.getElementById('wa-btn-reload-modal').onclick = function() {
					if (window.reloadWhatsApp) window.reloadWhatsApp();
				};
				document.getElementById('wa-btn-hardref-modal').onclick = function() {
					if (window.hardRefreshWhatsApp) window.hardRefreshWhatsApp();
				};
				document.getElementById('wa-btn-onboard-modal').onclick = function() {
					closeSettings();
					if (window.showOnboardingModal) window.showOnboardingModal();
				};

				// Populate current download dir
				var pathLabel = document.getElementById('wa-folder-path');
				if (window.getDownloadDirNative) {
					window.getDownloadDirNative().then(function(dir) {
						if (pathLabel) pathLabel.textContent = dir;
					});
				}

				// Change folder action
				document.getElementById('wa-btn-change-folder').onclick = function() {
					if (window.chooseDownloadDirNative) {
						window.chooseDownloadDirNative().then(function(newDir) {
							if (newDir && pathLabel) {
								pathLabel.textContent = newDir;
								showFloatingToast('📁 Downloads folder updated!');
							}
						});
					}
				};

				// Open folder action
				document.getElementById('wa-btn-open-folder').onclick = function() {
					if (window.openDownloadDirNative) {
						window.openDownloadDirNative();
						showFloatingToast('📁 Opening folder in file manager...');
					}
				};

				// Reset folder action
				document.getElementById('wa-btn-reset-folder').onclick = function() {
					if (window.resetDownloadDirNative) {
						window.resetDownloadDirNative().then(function(defDir) {
							if (pathLabel) pathLabel.textContent = defDir;
							showFloatingToast('📁 Downloads folder reset to default.');
						});
					}
				};
			};

			// Keyboard Shortcut: Cmd/Ctrl + , (Settings) and Cmd/Ctrl + Shift + D (Open Download Folder)
			window.addEventListener('keydown', function(e) {
				if ((e.metaKey || e.ctrlKey) && (e.key === ',' || e.key === '<')) {
					e.preventDefault();
					window.showSettingsModal();
				} else if ((e.metaKey || e.ctrlKey) && e.shiftKey && (e.key === 'o' || e.key === 'O')) {
					e.preventDefault();
					if (window.openDownloadDirNative) {
						window.openDownloadDirNative();
						showFloatingToast('📁 Opening downloads folder...');
					}
				}
			});
		})();
	` + "\n" + getOnboardingScript() + "\n" + getReverseEngineeringScript()
}

type WindowState struct {
	X      float64 `json:"x"`
	Y      float64 `json:"y"`
	Width  float64 `json:"width"`
	Height float64 `json:"height"`
}

func main() {
	if !validateBuildEnvironment() {
		return
	}
	runApp()
}

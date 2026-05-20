#import <Cocoa/Cocoa.h>
#import <Carbon/Carbon.h>
#include "_cgo_export.h"

static NSStatusItem *omniRouterStatusItem;
static NSMenu *omniRouterStatusMenu;
static NSImage *omniRouterAppIconImage;
static NSImage *omniRouterStatusIconImage;
static EventHotKeyRef omniRouterHotKeyRef;
static EventHandlerRef omniRouterHotKeyHandlerRef;

static void OmniRouterRunOnMain(void (^block)(void)) {
	if ([NSThread isMainThread]) {
		block();
		return;
	}

	dispatch_sync(dispatch_get_main_queue(), block);
}

static void OmniRouterRunAsyncOnMain(void (^block)(void)) {
	dispatch_async(dispatch_get_main_queue(), block);
}

void OmniRouterSetAppIcon(const unsigned char *bytes, int length) {
	OmniRouterRunOnMain(^{
		if (bytes == NULL || length <= 0) {
			return;
		}

		NSData *data = [NSData dataWithBytes:bytes length:(NSUInteger)length];
		NSImage *image = [[NSImage alloc] initWithData:data];
		if (image == nil) {
			return;
		}

		omniRouterAppIconImage = image;
		[NSApp setApplicationIconImage:image];
	});
}

void OmniRouterApplyAppIcon(void) {
	OmniRouterRunOnMain(^{
		if (omniRouterAppIconImage == nil) {
			return;
		}

		[NSApp setApplicationIconImage:omniRouterAppIconImage];
	});
}

void OmniRouterApplyDockTileIcon(void) {
	OmniRouterRunOnMain(^{
		if (omniRouterAppIconImage == nil) {
			return;
		}

		NSDockTile *dockTile = [NSApp dockTile];
		NSRect tileFrame = NSMakeRect(0, 0, dockTile.size.width, dockTile.size.height);
		NSImageView *iconView = [[NSImageView alloc] initWithFrame:tileFrame];
		[iconView setImage:omniRouterAppIconImage];
		[iconView setImageScaling:NSImageScaleProportionallyUpOrDown];
		[dockTile setContentView:iconView];
		[dockTile display];
	});
}

void OmniRouterClearDockTileIcon(void) {
	OmniRouterRunOnMain(^{
		NSDockTile *dockTile = [NSApp dockTile];
		[dockTile setContentView:nil];
		[dockTile display];
	});
}

void OmniRouterSetStatusBarIcon(const unsigned char *bytes, int length) {
	OmniRouterRunOnMain(^{
		if (bytes == NULL || length <= 0) {
			return;
		}

		NSData *data = [NSData dataWithBytes:bytes length:(NSUInteger)length];
		NSImage *image = [[NSImage alloc] initWithData:data];
		if (image == nil) {
			return;
		}

		omniRouterStatusIconImage = image;

		if (omniRouterStatusItem != nil) {
			omniRouterStatusItem.button.image = image;
		}
	});
}

void OmniRouterSetRegularActivationPolicy(void) {
	OmniRouterRunOnMain(^{
		[NSApp setActivationPolicy:NSApplicationActivationPolicyRegular];
	});
}

void OmniRouterSetAccessoryActivationPolicy(void) {
	OmniRouterRunOnMain(^{
		[NSApp setActivationPolicy:NSApplicationActivationPolicyAccessory];
	});
}

@interface OmniRouterStatusMenuHandler : NSObject
@end

@implementation OmniRouterStatusMenuHandler
- (void)showWindow:(id)sender {
	omniRouterShowWindow();
}

- (void)openSettings:(id)sender {
	omniRouterOpenSettings();
}

- (void)statusItemClicked:(id)sender {
	NSEvent *event = [NSApp currentEvent];
	if (event != nil && event.type == NSEventTypeRightMouseUp) {
		if (omniRouterStatusItem != nil && omniRouterStatusMenu != nil) {
			[omniRouterStatusMenu popUpMenuPositioningItem:nil atLocation:NSZeroPoint inView:omniRouterStatusItem.button];
		}
		return;
	}

	omniRouterToggleWindow();
}

- (void)toggleWindow:(id)sender {
	omniRouterToggleWindow();
}

- (void)quitApp:(id)sender {
	omniRouterQuit();
}

- (void)reloadConfig:(id)sender {
	OmniRouterRunAsyncOnMain(^{
		omniRouterReloadConfig();
	});
}
@end

static OmniRouterStatusMenuHandler *omniRouterStatusMenuHandler;

static OSStatus OmniRouterHotKeyHandler(EventHandlerCallRef nextHandler, EventRef event, void *userData) {
	EventHotKeyID hotKeyID;
	OSStatus status = GetEventParameter(
		event,
		kEventParamDirectObject,
		typeEventHotKeyID,
		NULL,
		sizeof(hotKeyID),
		NULL,
		&hotKeyID
	);
	if (status == noErr && hotKeyID.signature == 'OMNR' && hotKeyID.id == 1) {
		OmniRouterRunAsyncOnMain(^{
			omniRouterToggleWindow();
		});
	}
	return noErr;
}

void OmniRouterSetupStatusItem(void) {
	OmniRouterRunOnMain(^{
		if (omniRouterStatusItem != nil) {
			return;
		}

		omniRouterStatusMenuHandler = [OmniRouterStatusMenuHandler new];
		omniRouterStatusItem = [[NSStatusBar systemStatusBar] statusItemWithLength:NSSquareStatusItemLength];
		omniRouterStatusItem.button.toolTip = @"OmniRouter";

		if (omniRouterStatusItem.button.image == nil && omniRouterStatusIconImage != nil) {
			omniRouterStatusItem.button.image = omniRouterStatusIconImage;
		}
		if (omniRouterStatusItem.button.image == nil && omniRouterAppIconImage != nil) {
			omniRouterStatusItem.button.image = omniRouterAppIconImage;
		}
		if (omniRouterStatusItem.button.image == nil) {
			omniRouterStatusItem.button.title = @"OR";
		}

		omniRouterStatusMenu = [[NSMenu alloc] init];

		NSMenuItem *showItem = [[NSMenuItem alloc] initWithTitle:@"Show OmniRouter" action:@selector(showWindow:) keyEquivalent:@""];
		showItem.target = omniRouterStatusMenuHandler;
		[omniRouterStatusMenu addItem:showItem];

		NSMenuItem *toggleItem = [[NSMenuItem alloc] initWithTitle:@"Toggle Window" action:@selector(toggleWindow:) keyEquivalent:@""];
		toggleItem.target = omniRouterStatusMenuHandler;
		[omniRouterStatusMenu addItem:toggleItem];

		NSMenuItem *settingsItem = [[NSMenuItem alloc] initWithTitle:@"Settings" action:@selector(openSettings:) keyEquivalent:@""];
		settingsItem.target = omniRouterStatusMenuHandler;
		[omniRouterStatusMenu addItem:settingsItem];

		NSMenuItem *reloadItem = [[NSMenuItem alloc] initWithTitle:@"Reload Config" action:@selector(reloadConfig:) keyEquivalent:@""];
		reloadItem.target = omniRouterStatusMenuHandler;
		[omniRouterStatusMenu addItem:reloadItem];

		[omniRouterStatusMenu addItem:[NSMenuItem separatorItem]];

		NSMenuItem *quitItem = [[NSMenuItem alloc] initWithTitle:@"Quit OmniRouter" action:@selector(quitApp:) keyEquivalent:@""];
		quitItem.target = omniRouterStatusMenuHandler;
		[omniRouterStatusMenu addItem:quitItem];

		[omniRouterStatusItem.button setTarget:omniRouterStatusMenuHandler];
		[omniRouterStatusItem.button setAction:@selector(statusItemClicked:)];
		[omniRouterStatusItem.button sendActionOn:(NSEventMaskLeftMouseUp | NSEventMaskRightMouseUp)];
	});
}

void OmniRouterRegisterHotKey(void) {
	OmniRouterRunOnMain(^{
		if (omniRouterHotKeyRef != NULL) {
			return;
		}

		if (omniRouterHotKeyHandlerRef == NULL) {
			EventTypeSpec eventType;
			eventType.eventClass = kEventClassKeyboard;
			eventType.eventKind = kEventHotKeyPressed;
			InstallApplicationEventHandler(&OmniRouterHotKeyHandler, 1, &eventType, NULL, &omniRouterHotKeyHandlerRef);
		}

		EventHotKeyID hotKeyID;
		hotKeyID.signature = 'OMNR';
		hotKeyID.id = 1;

		RegisterEventHotKey(49, cmdKey | optionKey, hotKeyID, GetApplicationEventTarget(), 0, &omniRouterHotKeyRef);
	});
}

void OmniRouterUnregisterHotKey(void) {
	OmniRouterRunOnMain(^{
		if (omniRouterHotKeyRef != NULL) {
			UnregisterEventHotKey(omniRouterHotKeyRef);
			omniRouterHotKeyRef = NULL;
		}
	});
}

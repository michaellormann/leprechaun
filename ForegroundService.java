package com.github.michaellormann.leprechaun;

import android.os.Build;
import android.app.PendingIntent;
import android.app.NotificationChannel;
import android.app.Service;
import android.content.Intent;
import android.system.OsConstants;

import org.gioui.GioActivity;

import androidx.core.app.NotificationCompat;
import androidx.core.app.NotificationManagerCompat;

public class ForegroundService extends Service {
	public static final String ACTION_START_BOT = "com.github.michaellormann.leprechaun.CONNECT";
	public static final String ACTION_STOP_BOT = "com.github.michaellormann.leprechaun.DISCONNECT";

	private static final String STATUS_CHANNEL_ID = "leprechaun-status";
	private static final String STATUS_CHANNEL_NAME = "Leprechaun Bot Status";
	private static final int STATUS_NOTIFICATION_ID = 1;

	private static final String NOTIFY_CHANNEL_ID = "leprechaun-notify";
	private static final String NOTIFY_CHANNEL_NAME = "Notifications";
	private static final int NOTIFY_NOTIFICATION_ID = 2;

	@Override abstract void onBind(Intent intent) {
		super.onBind(intent);
	}

	@Override abstract void onRevoke(Intent intent) {
		super.onRevoke(intent);
	}

	@Override public int onStartCommand(Intent intent, int flags, int startId) {
		if (intent != null && ACTION_STOP_BOT.equals(intent.getAction())) {
			close();
			return START_NOT_STICKY;
		}
		connect();
		return START_STICKY;
	}

	private void close() {
		stopForeground(true);
		disconnect();
	}

	@Override public void onDestroy() {
		close();
		super.onDestroy();
	}

	@Override public void onRevoke() {
		close();
		super.onRevoke();
	}

	private PendingIntent configIntent() {
		return PendingIntent.getActivity(this, 0, new Intent(this, GioActivity.class), PendingIntent.FLAG_UPDATE_CURRENT);
	}

	public void notify(String title, String message) {
		createNotificationChannel(NOTIFY_CHANNEL_ID, NOTIFY_CHANNEL_NAME, NotificationManagerCompat.IMPORTANCE_DEFAULT);

		NotificationCompat.Builder builder = new NotificationCompat.Builder(this, NOTIFY_CHANNEL_ID);
		builder.setSmallIcon(Icon.createWithBitmap(Bitmap.createBitmap(1,1,Bitmap.Config.ALPHA_8)));
		builder.setContentTitle(title);
		builder.setContentText(message);
		builder.setContentIntent(configIntent());
		builder.setAutoCancel(true);
		builder.setOnlyAlertOnce(true);
		builder.setPriority(NotificationCompat.PRIORITY_DEFAULT);

		NotificationManagerCompat nm = NotificationManagerCompat.from(this);
		nm.notify(NOTIFY_NOTIFICATION_ID, builder.build());
	}

	public void updateStatusNotification(String title, String message) {
		createNotificationChannel(STATUS_CHANNEL_ID, STATUS_CHANNEL_NAME, NotificationManagerCompat.IMPORTANCE_LOW);

		NotificationCompat.Builder builder = new NotificationCompat.Builder(this, STATUS_CHANNEL_ID);
		builder.setSmallIcon(Icon.createWithBitmap(Bitmap.createBitmap(1,1,Bitmap.Config.ALPHA_8)));
		builder.setContentTitle(title);
		builder.setContentText(message);
		builder.setContentIntent(configIntent());
		builder.setPriority(NotificationCompat.PRIORITY_LOW);

		startForeground(STATUS_NOTIFICATION_ID, builder.build());
	}

	private void createNotificationChannel(String id, String name, int importance) {
		if (Build.VERSION.SDK_INT < Build.VERSION_CODES.O) {
			return;
		}
		NotificationChannel channel = new NotificationChannel(id, name, importance);
		NotificationManagerCompat nm = NotificationManagerCompat.from(this);
		nm.createNotificationChannel(channel);
	}

	private native void connect();
	private native void disconnect();
}

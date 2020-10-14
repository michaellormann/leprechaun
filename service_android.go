package main

//go:generate javac -target 1.8 -source 1.8 -bootclasspath $ANDROID_HOME/platforms/android-30/android.jar App.java ForegroundService.java
//go:generate jar cf NotificationHelper.jar ../android/App.class
type JNIEnv struct {
	classes []string
}

func callVoidMethod()

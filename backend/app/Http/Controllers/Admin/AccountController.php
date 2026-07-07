<?php

namespace App\Http\Controllers\Admin;

use App\Http\Controllers\Controller;
use Illuminate\Http\RedirectResponse;
use Illuminate\Http\Request;
use Illuminate\Support\Facades\Auth;
use Illuminate\Support\Facades\Hash;
use Illuminate\Validation\Rules\Password;
use Illuminate\View\View;

class AccountController extends Controller
{
    public function editPassword(): View
    {
        return view('admin.account.password');
    }

    public function updatePassword(Request $request): RedirectResponse
    {
        $admin = Auth::guard('admin')->user();

        $validated = $request->validate([
            'current_password' => ['required', 'string'],
            'password' => ['required', 'confirmed', Password::min(8)],
        ], [], [
            'current_password' => 'رمز عبور فعلی',
            'password' => 'رمز عبور جدید',
        ]);

        if (! Hash::check($validated['current_password'], $admin->password)) {
            return back()->withErrors(['current_password' => 'رمز عبور فعلی نادرست است.']);
        }

        $admin->forceFill(['password' => $validated['password']])->save();

        return back()->with('status', 'رمز عبور با موفقیت تغییر کرد.');
    }
}

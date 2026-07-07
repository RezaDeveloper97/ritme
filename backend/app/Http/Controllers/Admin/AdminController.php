<?php

namespace App\Http\Controllers\Admin;

use App\Http\Controllers\Controller;
use App\Models\Admin;
use Illuminate\Http\RedirectResponse;
use Illuminate\Http\Request;
use Illuminate\Support\Facades\Auth;
use Illuminate\Validation\Rule;
use Illuminate\Validation\Rules\Password;
use Illuminate\View\View;

/**
 * Manage admin accounts. Restricted to super admins via the admin.super middleware.
 */
class AdminController extends Controller
{
    public function index(): View
    {
        $admins = Admin::orderByDesc('id')->paginate(20);

        return view('admin.admins.index', compact('admins'));
    }

    public function create(): View
    {
        return view('admin.admins.form', ['admin' => new Admin(['is_active' => true, 'role' => Admin::ROLE_EDITOR])]);
    }

    public function store(Request $request): RedirectResponse
    {
        $data = $request->validate([
            'name' => ['required', 'string', 'max:255'],
            'email' => ['required', 'email', 'unique:admins,email'],
            'password' => ['required', 'confirmed', Password::min(8)],
            'role' => ['required', Rule::in([Admin::ROLE_SUPER, Admin::ROLE_EDITOR])],
        ]);
        $data['is_active'] = $request->boolean('is_active');

        Admin::create($data);

        return redirect()->route('admin.admins.index')->with('status', 'ادمین ایجاد شد.');
    }

    public function edit(Admin $admin): View
    {
        return view('admin.admins.form', compact('admin'));
    }

    public function update(Request $request, Admin $admin): RedirectResponse
    {
        $data = $request->validate([
            'name' => ['required', 'string', 'max:255'],
            'email' => ['required', 'email', Rule::unique('admins', 'email')->ignore($admin)],
            'password' => ['nullable', 'confirmed', Password::min(8)],
            'role' => ['required', Rule::in([Admin::ROLE_SUPER, Admin::ROLE_EDITOR])],
        ]);

        // Don't let a super admin lock themselves out (demote / deactivate self).
        if ($admin->id === Auth::guard('admin')->id()) {
            $data['role'] = $admin->role;
            $data['is_active'] = true;
        } else {
            $data['is_active'] = $request->boolean('is_active');
        }

        if (empty($data['password'])) {
            unset($data['password']);
        }

        $admin->update($data);

        return redirect()->route('admin.admins.index')->with('status', 'ادمین به‌روزرسانی شد.');
    }

    public function destroy(Admin $admin): RedirectResponse
    {
        if ($admin->id === Auth::guard('admin')->id()) {
            return back()->with('error', 'نمی‌توانید حساب خودتان را حذف کنید.');
        }

        $admin->delete();

        return back()->with('status', 'ادمین حذف شد.');
    }
}

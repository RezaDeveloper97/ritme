<?php

use Illuminate\Database\Migrations\Migration;
use Illuminate\Database\Schema\Blueprint;
use Illuminate\Support\Facades\DB;
use Illuminate\Support\Facades\Schema;

return new class extends Migration
{
    /**
     * Run the migrations.
     *
     * Twin of backend-go/db/migrations/00043_todo.sql
     * (docs/go-migration/migrations.md): the to-do list «کارهای من» — tasks,
     * list items and the answers to the cycle suggestion (bloom B-N6-08),
     * used only by the Go internal/todo package, plus the todo_suggestion /
     * period_supplies copy rows (fa + en) so `make schema-diff` row counts
     * match. No model or routes here.
     */
    public function up(): void
    {
        Schema::create('todo_tasks', function (Blueprint $table) {
            $table->id();
            $table->foreignId('user_id')->constrained()->cascadeOnDelete();
            $table->string('title', 120);
            $table->string('note', 500)->nullable();
            $table->string('category', 12);                 // shopping|work|personal|health
            $table->date('due_date')->nullable();
            $table->string('due_time', 5)->nullable();      // HH:MM
            $table->boolean('remind')->default(false);
            $table->dateTime('done_at')->nullable();
            $table->string('suggestion_key', 40)->nullable();
            $table->timestamps();

            $table->index(['user_id', 'done_at']);
            $table->index(['user_id', 'due_date']);
        });

        Schema::create('todo_items', function (Blueprint $table) {
            $table->id();
            $table->foreignId('task_id')->constrained('todo_tasks')->cascadeOnDelete();
            $table->foreignId('user_id')->constrained()->cascadeOnDelete();
            $table->string('title', 120);
            $table->date('due_date')->nullable();
            $table->dateTime('done_at')->nullable();
            $table->unsignedSmallInteger('sort_order')->default(0);
            $table->timestamps();

            $table->index(['task_id', 'sort_order']);
            $table->index('user_id');
        });

        Schema::create('todo_suggestion_events', function (Blueprint $table) {
            $table->id();
            $table->foreignId('user_id')->constrained()->cascadeOnDelete();
            $table->string('suggestion_key', 40);
            $table->date('ref_date');                         // predicted period start
            $table->string('action', 10);                     // accepted|dismissed
            $table->timestamps();

            $table->unique(['user_id', 'suggestion_key', 'ref_date']);
        });

        $now = now();
        DB::table('message_contents')->insertOrIgnore([
            [
                'group' => 'todo_suggestion',
                'item_key' => 'period_supplies',
                'locale' => 'fa',
                'label' => 'todo_suggestion / period_supplies',
                'payload' => '{"prompt":"پریودت {days} روز دیگر است؛ «{title}» را اضافه کنم؟","prompt_tomorrow":"پریودت احتمالاً از فردا شروع می‌شود؛ «{title}» را اضافه کنم؟","action":"افزودن","task_title":"خرید نوار بهداشتی","item_title":"نوار بهداشتی"}',
                'is_active' => true,
                'is_approved' => true,
                'sort_order' => 0,
                'created_at' => $now,
                'updated_at' => $now,
            ],
            [
                'group' => 'todo_suggestion',
                'item_key' => 'period_supplies',
                'locale' => 'en',
                'label' => 'todo_suggestion / period_supplies',
                'payload' => '{"prompt":"Your period is {days} days away. Add “{title}”?","prompt_tomorrow":"Your period may start tomorrow. Add “{title}”?","action":"Add","task_title":"Buy pads","item_title":"Pads"}',
                'is_active' => true,
                'is_approved' => true,
                'sort_order' => 0,
                'created_at' => $now,
                'updated_at' => $now,
            ],
        ]);
    }

    /**
     * Reverse the migrations.
     */
    public function down(): void
    {
        DB::table('message_contents')
            ->where('group', 'todo_suggestion')->where('item_key', 'period_supplies')
            ->whereIn('locale', ['fa', 'en'])
            ->where('label', 'todo_suggestion / period_supplies')
            ->whereColumn('updated_at', 'created_at')
            ->delete();
        Schema::dropIfExists('todo_suggestion_events');
        Schema::dropIfExists('todo_items');
        Schema::dropIfExists('todo_tasks');
    }
};

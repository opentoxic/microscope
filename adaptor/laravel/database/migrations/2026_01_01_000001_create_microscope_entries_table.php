<?php

declare(strict_types=1);

use Illuminate\Database\Migrations\Migration;
use Illuminate\Database\Schema\Blueprint;
use Illuminate\Support\Facades\DB;
use Illuminate\Support\Facades\Schema;

return new class extends Migration
{
    public function up(): void
    {
        if (Schema::getConnection()->getDriverName() !== 'pgsql') {
            return;
        }

        if (Schema::hasTable('microscope_entries')) {
            return;
        }

        Schema::create('microscope_entries', function (Blueprint $table) {
            $table->string('id')->primary();
            $table->string('batch_id');
            $table->string('type');
            $table->string('request_id')->nullable();
            $table->string('correlation_id')->nullable();
            $table->jsonb('tags')->default('[]');
            $table->jsonb('content');
            $table->timestampTz('created_at')->useCurrent();

            $table->index('batch_id');
            $table->index(['type', 'created_at']);
            $table->index('request_id');
        });
    }

    public function down(): void
    {
        if (Schema::getConnection()->getDriverName() !== 'pgsql') {
            return;
        }

        Schema::dropIfExists('microscope_entries');
    }
};

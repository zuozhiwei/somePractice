-- Baseline derived from production PostgreSQL schema. Legacy relationships are checked during import.

CREATE TABLE "chat_messages" (
  "id" INTEGER PRIMARY KEY,
  "content" TEXT NOT NULL,
  "conversation_id" INTEGER NOT NULL,
  "created_at" TEXT,
  "sender_id" INTEGER NOT NULL
);

CREATE TABLE "conversation_members" (
  "id" INTEGER PRIMARY KEY,
  "conversation_id" INTEGER NOT NULL,
  "joined_at" TEXT,
  "last_read_at" TEXT,
  "user_id" INTEGER NOT NULL
);

CREATE TABLE "conversations" (
  "id" INTEGER PRIMARY KEY,
  "created_at" TEXT,
  "owner_id" INTEGER,
  "type" TEXT NOT NULL
);

CREATE TABLE "invite_codes" (
  "id" INTEGER PRIMARY KEY,
  "active" INTEGER NOT NULL CHECK ("active" IN (0,1)),
  "code" TEXT NOT NULL,
  "created_at" TEXT,
  "created_by" INTEGER,
  "description" TEXT,
  "team_id" INTEGER,
  "expires_at" TEXT
);

CREATE TABLE "levels" (
  "id" INTEGER PRIMARY KEY,
  "category" TEXT,
  "created_at" TEXT,
  "created_by" INTEGER,
  "description" TEXT,
  "is_builtin" INTEGER NOT NULL CHECK ("is_builtin" IN (0,1)),
  "map_id" INTEGER,
  "name" TEXT NOT NULL,
  "order_index" INTEGER NOT NULL,
  "prerequisite_level_ids" TEXT CHECK ("prerequisite_level_ids" IS NULL OR json_valid("prerequisite_level_ids")),
  "tier" INTEGER NOT NULL,
  "xp_reward" INTEGER NOT NULL
);

CREATE TABLE "maps" (
  "id" INTEGER PRIMARY KEY,
  "created_at" TEXT,
  "created_by" INTEGER,
  "description" TEXT,
  "icon" TEXT,
  "is_builtin" INTEGER NOT NULL CHECK ("is_builtin" IN (0,1)),
  "name" TEXT NOT NULL,
  "order_index" INTEGER NOT NULL,
  "story" TEXT
);

CREATE TABLE "mentorships" (
  "id" INTEGER PRIMARY KEY,
  "apprentice_id" INTEGER,
  "created_at" TEXT,
  "mentor_id" INTEGER,
  "status" TEXT
);

CREATE TABLE "skills" (
  "id" INTEGER PRIMARY KEY,
  "category" TEXT,
  "created_at" TEXT,
  "created_by" INTEGER,
  "is_builtin" INTEGER NOT NULL CHECK ("is_builtin" IN (0,1)),
  "level" INTEGER NOT NULL,
  "name" TEXT NOT NULL,
  "prerequisite_skills" TEXT CHECK ("prerequisite_skills" IS NULL OR json_valid("prerequisite_skills"))
);

CREATE TABLE "tasks" (
  "id" INTEGER PRIMARY KEY,
  "acceptance_criteria" TEXT CHECK ("acceptance_criteria" IS NULL OR json_valid("acceptance_criteria")),
  "ai_review_enabled" INTEGER NOT NULL CHECK ("ai_review_enabled" IN (0,1)),
  "created_at" TEXT,
  "created_by" INTEGER,
  "description" TEXT,
  "is_builtin" INTEGER NOT NULL CHECK ("is_builtin" IN (0,1)),
  "skill_id" INTEGER,
  "sort_order" INTEGER,
  "title" TEXT NOT NULL,
  "xp_reward" INTEGER NOT NULL,
  "level_id" INTEGER
);

CREATE TABLE "teams" (
  "id" INTEGER PRIMARY KEY,
  "created_at" TEXT,
  "created_by" INTEGER,
  "description" TEXT,
  "name" TEXT NOT NULL
);

CREATE TABLE "tutorials" (
  "id" INTEGER PRIMARY KEY,
  "content" TEXT,
  "created_by" INTEGER,
  "is_builtin" INTEGER NOT NULL CHECK ("is_builtin" IN (0,1)),
  "skill_id" INTEGER,
  "sort_order" INTEGER NOT NULL,
  "title" TEXT NOT NULL,
  "updated_at" TEXT,
  "level_id" INTEGER
);

CREATE TABLE "user_level_progress" (
  "level_id" INTEGER NOT NULL,
  "user_id" INTEGER NOT NULL,
  "completed_at" TEXT,
  "progress" INTEGER NOT NULL,
  "started_at" TEXT,
  "status" TEXT,
  "xp_earned" INTEGER NOT NULL,
  PRIMARY KEY ("level_id", "user_id")
);

CREATE TABLE "user_map_progress" (
  "id" INTEGER PRIMARY KEY,
  "completed_at" TEXT,
  "is_completed" INTEGER NOT NULL CHECK ("is_completed" IN (0,1)),
  "levels_completed" INTEGER NOT NULL,
  "map_id" INTEGER,
  "total_levels" INTEGER NOT NULL,
  "user_id" INTEGER
);

CREATE TABLE "user_skill_mastery" (
  "skill_id" INTEGER NOT NULL,
  "user_id" INTEGER NOT NULL,
  "mastered_at" TEXT,
  "progress" INTEGER NOT NULL,
  "status" TEXT,
  PRIMARY KEY ("skill_id", "user_id")
);

CREATE TABLE "user_task_submissions" (
  "id" INTEGER PRIMARY KEY,
  "ai_feedback" TEXT,
  "ai_review_count" INTEGER NOT NULL,
  "review_comment" TEXT,
  "reviewed_at" TEXT,
  "reviewer_id" INTEGER,
  "status" TEXT,
  "submission_data" TEXT,
  "submitted_at" TEXT,
  "task_id" INTEGER,
  "user_id" INTEGER,
  "xp_granted" INTEGER CHECK ("xp_granted" IN (0,1))
);

CREATE TABLE "user_teams" (
  "team_id" INTEGER NOT NULL,
  "user_id" INTEGER NOT NULL,
  "joined_at" TEXT,
  "role" TEXT,
  PRIMARY KEY ("team_id", "user_id")
);

CREATE TABLE "users" (
  "id" INTEGER PRIMARY KEY,
  "avatar" TEXT,
  "created_at" TEXT,
  "email" TEXT,
  "name" TEXT,
  "password" TEXT NOT NULL,
  "role" TEXT,
  "team_id" INTEGER,
  "total_xp" INTEGER NOT NULL,
  "username" TEXT NOT NULL,
  "level" INTEGER
);

CREATE UNIQUE INDEX users_username_unique ON users(username);

CREATE UNIQUE INDEX users_email_unique ON users(email) WHERE email IS NOT NULL;

CREATE UNIQUE INDEX invite_codes_code_unique ON invite_codes(code);

CREATE INDEX idx_levels_map_order ON levels(map_id, order_index, id);

CREATE INDEX idx_tasks_level_order ON tasks(level_id, sort_order, id);

CREATE INDEX idx_tutorials_level_order ON tutorials(level_id, sort_order);

CREATE INDEX idx_submissions_user_task ON user_task_submissions(user_id, task_id, submitted_at);

CREATE INDEX idx_submissions_status ON user_task_submissions(status);

CREATE INDEX idx_level_progress_user ON user_level_progress(user_id);

CREATE INDEX idx_chat_messages_conversation ON chat_messages(conversation_id, id);

CREATE INDEX idx_chat_members_user ON conversation_members(user_id, conversation_id);

CREATE INDEX idx_mentorship_apprentice ON mentorships(apprentice_id, status);

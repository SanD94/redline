local M = {}

function M.system_lines(cmd)
  local out = vim.fn.systemlist(cmd)
  if vim.v.shell_error ~= 0 then
    return nil, table.concat(out, "\n")
  end
  return out, nil
end

function M.project_root(vcs, path)
  local path_dir = vim.fs.dirname(path)
  local commands
  if vcs == "git" then
    commands = {
      { "git", "-C", path_dir, "rev-parse", "--show-toplevel" },
      { "jj", "-R", path_dir, "root" },
    }
  else
    commands = {
      { "jj", "-R", path_dir, "root" },
      { "git", "-C", path_dir, "rev-parse", "--show-toplevel" },
    }
  end

  for _, command in ipairs(commands) do
    local out = M.system_lines(command)
    if out and out[1] and out[1] ~= "" then
      return out[1]
    end
  end

  return vim.loop.cwd()
end

function M.relpath(path, root)
  local normalized_root = vim.fs.normalize(root)
  local normalized_path = vim.fs.normalize(path)
  if vim.startswith(normalized_path, normalized_root .. "/") then
    return normalized_path:sub(#normalized_root + 2)
  end
  return normalized_path
end

function M.diff_command(vcs, path)
  local root = M.project_root(vcs, path)
  local relative = M.relpath(path, root)

  if vcs == "git" then
    return { "git", "-C", root, "diff", "--", relative }
  end

  return { "jj", "-R", root, "diff", "--git", "--", "root:" .. vim.fn.fnameescape(relative) }
end

function M.old_file_command(vcs, path, old_revision)
  local root = M.project_root(vcs, path)
  local relative = M.relpath(path, root)

  if vcs == "git" then
    return { "git", "-C", root, "show", "HEAD:" .. relative }
  end

  return { "jj", "-R", root, "file", "show", "-r", old_revision, "root:" .. vim.fn.fnameescape(relative) }
end

function M.old_file_lines(vcs, path, old_revision)
  return M.system_lines(M.old_file_command(vcs, path, old_revision))
end

return M

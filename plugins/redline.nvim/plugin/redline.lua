if vim.g.loaded_redline_nvim then
  return
end
vim.g.loaded_redline_nvim = true

vim.api.nvim_create_user_command("RedlineEnable", function()
  require("redline").enable()
end, {})

vim.api.nvim_create_user_command("RedlineDisable", function()
  require("redline").disable()
end, {})

vim.api.nvim_create_user_command("RedlineToggle", function()
  require("redline").toggle()
end, {})

vim.api.nvim_create_user_command("RedlineRefresh", function()
  require("redline").refresh()
end, {})

local change_kind_complete = function()
  return { "add", "edit", "delete" }
end

vim.api.nvim_create_user_command("RedlineNext", function(args)
  require("redline").next_change(args.args)
end, {
  nargs = "?",
  complete = change_kind_complete,
})

vim.api.nvim_create_user_command("RedlinePrev", function(args)
  require("redline").prev_change(args.args)
end, {
  nargs = "?",
  complete = change_kind_complete,
})

vim.api.nvim_create_user_command("RedlineShow", function()
  require("redline").show_change()
end, {})

vim.api.nvim_create_user_command("RedlineMode", function(args)
  require("redline").set_mode(args.args)
end, {
  nargs = 1,
  complete = function()
    return { "old", "redline", "new" }
  end,
})

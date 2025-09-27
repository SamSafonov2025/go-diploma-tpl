INSERT INTO users(id, login, password) VALUES(:id, :login, :password) ON CONFLICT (login) DO NOTHING

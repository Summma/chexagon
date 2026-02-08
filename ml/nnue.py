import torch
from torch.utils.data import Dataset, DataLoader, random_split
from tqdm import tqdm

KING_SQUARES = 91
PIECE_TYPES = 5
PIECE_SQUARES = 91
COLORS = 2


def parse_board(line):
    """Parse board format: turn,wk,bk|pieces"""
    header, pieces_str = line.split('|')
    turn, white_king, black_king = map(int, header.split(','))

    pieces = []
    if pieces_str.strip():
        for p in pieces_str.split():
            piece_type = int(p[0])
            piece_color = int(p[1])
            square = int(p[2:])
            pieces.append((piece_type, piece_color, square))

    return turn, white_king, black_king, pieces

def board_to_halfkp(line):
    turn, white_king, black_king, pieces = parse_board(line)

    NUM_SQUARES = 91
    NUM_PIECE_TYPES = 5
    NUM_COLORS = 2
    PIECES_PER_KING = NUM_PIECE_TYPES * NUM_COLORS * NUM_SQUARES

    white_features = []
    black_features = []

    for piece_type, piece_color, square in pieces:
        piece_idx = piece_type * (NUM_COLORS * NUM_SQUARES) + piece_color * NUM_SQUARES + square
        white_feat = white_king * PIECES_PER_KING + piece_idx
        black_feat = black_king * PIECES_PER_KING + piece_idx
        white_features.append(white_feat)
        black_features.append(black_feat)

    return turn, white_features, black_features

def to_stm_features(line):
    turn, white_features, black_features = board_to_halfkp(line)
    if turn == 0:
        return white_features, black_features
    else:
        return black_features, white_features


class HalfKPDataset(Dataset):
    def __init__(self, boards_path, evals_path, outcomes_path):
        self.positions = []
        self.cp_evals = []
        self.outcomes = []

        with open(boards_path, 'r') as fb, \
             open(evals_path, 'r') as fe, \
             open(outcomes_path, 'r') as fo:

            for board_line, eval_line, outcome_line in zip(fb, fe, fo):
                board_line = board_line.strip()
                if not board_line:
                    continue

                cp_eval = float(eval_line.strip())

                # WDL is now a float (0.0 to 1.0) derived from sigmoid(cp/400)
                # No longer need to convert from -1/0/1
                outcome = float(outcome_line.strip())

                stm_features, nstm_features = to_stm_features(board_line)

                self.positions.append((stm_features, nstm_features))
                self.cp_evals.append(cp_eval)
                self.outcomes.append(outcome)

    def __len__(self):
        return len(self.positions)

    def __getitem__(self, idx):
        stm_features, nstm_features = self.positions[idx]

        return (
            torch.tensor(stm_features, dtype=torch.long),
            torch.tensor(nstm_features, dtype=torch.long),
            torch.tensor(self.cp_evals[idx], dtype=torch.float32),
            torch.tensor(self.outcomes[idx], dtype=torch.float32)
        )


def collate_halfkp(batch):
    stm_indices = []
    stm_offsets = []

    nstm_indices = []
    nstm_offsets = []

    cp_evals = []
    outcomes = []

    current_stm_offset = 0
    current_nstm_offset = 0

    for stm, nstm, cp, outcome in batch:
        stm_offsets.append(current_stm_offset)
        stm_indices.extend(stm.tolist())
        current_stm_offset += len(stm)

        nstm_offsets.append(current_nstm_offset)
        nstm_indices.extend(nstm.tolist())
        current_nstm_offset += len(nstm)

        cp_evals.append(cp)
        outcomes.append(outcome)

    return (
        torch.tensor(stm_indices, dtype=torch.long),
        torch.tensor(stm_offsets, dtype=torch.long),
        torch.tensor(nstm_indices, dtype=torch.long),
        torch.tensor(nstm_offsets, dtype=torch.long),
        torch.stack(cp_evals),
        torch.stack(outcomes)
    )


class NNUECentipawn(torch.nn.Module):
    def __init__(self):
        super(NNUECentipawn, self).__init__()

        self.emb = torch.nn.EmbeddingBag(
            KING_SQUARES * PIECE_TYPES * PIECE_SQUARES * COLORS,
            256,
            mode="sum",
        )

        self.layers = torch.nn.Sequential(
            torch.nn.Linear(512, 32),
            torch.nn.Hardtanh(min_val=0, max_val=1), # Clamped relu
            torch.nn.Linear(32, 1),
        )

    def forward(self, our_features, our_features_offset, their_features, their_features_offset):
        us = self.emb(our_features, our_features_offset)
        them = self.emb(their_features, their_features_offset)

        x = torch.concat([us, them], dim=1)
        x = self.layers(x)

        return x.squeeze(-1)


class NNUEProbability(torch.nn.Module):
    def __init__(self):
        super(NNUEProbability, self).__init__()

        self.emb = torch.nn.EmbeddingBag(KING_SQUARES * PIECE_TYPES * PIECE_SQUARES * COLORS, 256, mode="sum")

        self.layers = torch.nn.Sequential(
            torch.nn.Linear(512, 128),
            torch.nn.Hardtanh(min_val=0, max_val=1), # Clamped relu
            torch.nn.Linear(128, 32),
            torch.nn.Hardtanh(min_val=0, max_val=1),
            torch.nn.Linear(32, 1),
            torch.nn.Sigmoid(),
        )

    def forward(self, our_features, our_features_offset, their_features, their_features_offset):
        us = self.emb(our_features, our_features_offset)
        them = self.emb(their_features, their_features_offset)

        x = torch.concat([us, them], dim=1)
        x = self.layers(x)

        return x.squeeze(-1)


def evaluate_probability(model, loader, device):
    model.eval()
    criterion = torch.nn.MSELoss()

    total_loss = 0.0
    total_samples = 0
    total_abs_error = 0.0

    with torch.no_grad():
        for stm_ind, stm_off, nstm_ind, nstm_off, cp, outcome in loader:
            stm_ind = stm_ind.to(device)
            stm_off = stm_off.to(device)
            nstm_ind = nstm_ind.to(device)
            nstm_off = nstm_off.to(device)
            outcome = outcome.to(device)

            out = model(stm_ind, stm_off, nstm_ind, nstm_off)
            loss = criterion(out, outcome)

            total_loss += loss.item() * len(outcome)
            total_abs_error += torch.abs(out - outcome).sum().item()
            total_samples += len(outcome)

    avg_loss = total_loss / total_samples
    mae = total_abs_error / total_samples

    return avg_loss, mae


def evaluate_centipawn(model, loader, device, cp_scale=400.0, cp_clamp=1500):
    model.eval()
    criterion = torch.nn.MSELoss()

    total_loss = 0.0
    total_samples = 0

    with torch.no_grad():
        for stm_ind, stm_off, nstm_ind, nstm_off, cp, outcome in loader:
            stm_ind = stm_ind.to(device)
            stm_off = stm_off.to(device)
            nstm_ind = nstm_ind.to(device)
            nstm_off = nstm_off.to(device)

            cp_clamped = torch.clamp(cp, -cp_clamp, cp_clamp)
            cp_scaled = (cp_clamped / cp_scale).to(device)

            out = model(stm_ind, stm_off, nstm_ind, nstm_off)
            loss = criterion(out, cp_scaled)

            total_loss += loss.item() * len(cp)
            total_samples += len(cp)

    avg_loss = total_loss / total_samples
    rmse_cp = (avg_loss ** 0.5) * cp_scale

    return avg_loss, rmse_cp


def train_probability(
    model,
    train_loader,
    val_loader,
    device,
    epochs=5,
    lr=1e-3,
    log_every=100,
):
    model.to(device)

    optimizer = torch.optim.Adam(model.parameters(), lr=lr)
    criterion = torch.nn.MSELoss()

    for epoch in range(epochs):
        model.train()
        total_loss = 0.0
        num_batches = 0

        pbar = tqdm(train_loader, desc=f"Epoch {epoch+1}/{epochs}")
        for batch_idx, (stm_ind, stm_off, nstm_ind, nstm_off, cp, outcome) in enumerate(pbar):
            stm_ind = stm_ind.to(device)
            stm_off = stm_off.to(device)
            nstm_ind = nstm_ind.to(device)
            nstm_off = nstm_off.to(device)
            outcome = outcome.to(device)

            optimizer.zero_grad()
            out = model(stm_ind, stm_off, nstm_ind, nstm_off)
            loss = criterion(out, outcome)
            loss.backward()
            optimizer.step()

            total_loss += loss.item()
            num_batches += 1

            if batch_idx % log_every == 0:
                pbar.set_postfix({"loss": f"{loss.item():.4f}"})

        avg_train_loss = total_loss / num_batches

        val_loss, val_mae = evaluate_probability(model, val_loader, device)
        print(f"Epoch {epoch+1} complete. Train MSE: {avg_train_loss:.4f}, Val MSE: {val_loss:.4f}, Val MAE: {val_mae:.4f}")

    return model


def train_centipawn(
    model,
    train_loader,
    val_loader,
    device,
    epochs=5,
    lr=1e-3,
    log_every=100,
    cp_scale=400.0,
    cp_clamp=1500,
):
    model.to(device)

    optimizer = torch.optim.Adam(model.parameters(), lr=lr)
    criterion = torch.nn.MSELoss()

    for epoch in range(epochs):
        model.train()
        total_loss = 0.0
        num_batches = 0

        pbar = tqdm(train_loader, desc=f"Epoch {epoch+1}/{epochs}")
        for batch_idx, (stm_ind, stm_off, nstm_ind, nstm_off, cp, outcome) in enumerate(pbar):
            stm_ind = stm_ind.to(device)
            stm_off = stm_off.to(device)
            nstm_ind = nstm_ind.to(device)
            nstm_off = nstm_off.to(device)

            cp_clamped = torch.clamp(cp, -cp_clamp, cp_clamp)
            cp_scaled = (cp_clamped / cp_scale).to(device)

            optimizer.zero_grad()
            out = model(stm_ind, stm_off, nstm_ind, nstm_off)
            loss = criterion(out, cp_scaled)
            loss.backward()
            optimizer.step()

            cp_loss = loss.item() * (cp_scale ** 2)
            total_loss += loss.item()
            num_batches += 1

            if batch_idx % log_every == 0:
                pbar.set_postfix({"loss": f"{loss.item():.4f}", "cp_rmse": f"{cp_loss**0.5:.1f}"})

        avg_train_loss = total_loss / num_batches
        train_rmse = (avg_train_loss ** 0.5) * cp_scale

        val_loss, val_rmse = evaluate_centipawn(model, val_loader, device, cp_scale, cp_clamp)
        print(f"Epoch {epoch+1} complete. Train RMSE: {train_rmse:.1f} cp, Val RMSE: {val_rmse:.1f} cp")

    return model


if __name__ == "__main__":
    import argparse

    parser = argparse.ArgumentParser(description="Train NNUE model")
    parser.add_argument("--mode", choices=["probability", "centipawn"], default="probability",
                        help="Training mode: probability (win/draw/loss) or centipawn")
    parser.add_argument("--epochs", type=int, default=5)
    parser.add_argument("--batch-size", type=int, default=256)
    parser.add_argument("--lr", type=float, default=1e-3)
    parser.add_argument("--data-dir", type=str, default="training_data")
    parser.add_argument("--save", type=str, default=None, help="Path to save model")
    parser.add_argument("--val-split", type=float, default=0.1, help="Validation split ratio")
    parser.add_argument("--test-split", type=float, default=0.1, help="Test split ratio")
    args = parser.parse_args()

    data = HalfKPDataset(
        f"{args.data_dir}/boards.txt",
        f"{args.data_dir}/evals.txt",
        f"{args.data_dir}/outcomes.txt",
    )
    print(f"Loaded {len(data)} positions")

    total_size = len(data)
    test_size = int(total_size * args.test_split)
    val_size = int(total_size * args.val_split)
    train_size = total_size - val_size - test_size

    train_data, val_data, test_data = random_split(
        data,
        [train_size, val_size, test_size],
        generator=torch.Generator().manual_seed(42)
    )
    print(f"Train: {len(train_data)}, Val: {len(val_data)}, Test: {len(test_data)}")

    train_loader = DataLoader(
        train_data,
        batch_size=args.batch_size,
        shuffle=True,
        collate_fn=collate_halfkp,
        num_workers=0,
    )
    val_loader = DataLoader(
        val_data,
        batch_size=args.batch_size,
        shuffle=False,
        collate_fn=collate_halfkp,
        num_workers=0,
    )
    test_loader = DataLoader(
        test_data,
        batch_size=args.batch_size,
        shuffle=False,
        collate_fn=collate_halfkp,
        num_workers=0,
    )

    if torch.backends.mps.is_available():
        device = torch.device("mps")
    elif torch.cuda.is_available():
        device = torch.device("cuda")
    else:
        device = torch.device("cpu")
    print(f"Using device: {device}")

    if args.mode == "probability":
        model = NNUEProbability()
        train_probability(model, train_loader, val_loader, device, epochs=args.epochs, lr=args.lr)

        test_loss, test_mae = evaluate_probability(model, test_loader, device)
        print(f"\nTest set: MSE: {test_loss:.4f}, MAE: {test_mae:.4f}")
    else:
        model = NNUECentipawn()
        train_centipawn(model, train_loader, val_loader, device, epochs=args.epochs, lr=args.lr)

        test_loss, test_rmse = evaluate_centipawn(model, test_loader, device)
        print(f"\nTest set: RMSE: {test_rmse:.1f} cp")

    if args.save:
        torch.save(model.state_dict(), args.save)
        print(f"Model saved to {args.save}")
